# Match result updates and group-stage sync

This document describes the **end-to-end flow** when an administrator records or resets a match result: HTTP routes, validation, persistence, group standings recalculation, promotion of **1st and 2nd** place into the knockout bracket, and promotion of **best third-placed** teams after the full group stage is complete.

---

## Admin API surface

All routes require:

1. **`Auth` middleware** — valid JWT (`Authorization: Bearer …`).
2. **`RequireAdminRole`** — authenticated user’s `role` must be admin. Otherwise **403**.

| Method | Path | Purpose |
| ------ | ---- | ------- |
| `POST` | `/api/admin/matches/{id}/result` | Set **one** match to finished with scores; then run full group-stage sync. |
| `POST` | `/api/admin/matches/results` | Same as above for **many** matches in one request body. |
| `DELETE` | `/api/admin/matches/{id}/result` | Clear result (scores, winner, status back to scheduled); then run full group-stage sync. |
| `POST` | `/api/admin/standings/recalculate` | re-runs sync from current DB state (`SyncGroupStageOutcomes`). |
| `POST` | `/api/admin/standings/third-place/resolve` | Manual resolution when third-place promotion is in **conflict** state. |

---

## Request bodies and validation

### `POST /api/admin/matches/{id}/result`

- **Path:** `{id}` is the match id (integer). Bad id → **400**.
- **Body:** both scores are required; each is 0–99.

```json
{ "home_score": 2, "away_score": 1 }
```

- **Validation:** bad input → **400** with `"error": "validation failed"` and a **`details`** map (field name → short message, e.g. `"home_score": "home_score is required"`). Omitted scores are rejected.
- **Winner / status:** not sent by the client. The match is saved as **finished**; the winner is set from the scores (draw → no winner). See **Persistence** below.

### `POST /api/admin/matches/results`

- **Body:** `dtos.BulkUpdateMatchesResultDto`:

```json
{
  "matches": [
    { "id": 1, "home_score": 2, "away_score": 1 },
    { "id": 2, "home_score": 0, "away_score": 0 }
  ]
}
```

- **Validation:** `matches` must be a non-empty array. Each row needs **`id`**, **`home_score`**, and **`away_score`** (same rules as the single-match body). Errors use the same **400** + **`details`** shape as above.
- Each row is saved as **finished**; winner comes from the scores.

### `DELETE /api/admin/matches/{id}/result`

- No body. Same path validation as `POST` for `{id}`.
- Destructive: clears scores and winner and sets status to **`scheduled`** (`MatchRepository.ResetMatchResult`). Frontends should confirm with the user before calling (not enforced server-side).

### `POST /api/admin/standings/third-place/resolve`

Use when the last sync returned **`promotion_outcome.status == "conflict"`**. Send **exactly eight** distinct FIFA codes for the third-placed teams that should advance. The server checks they are a **subset of the current conflict `candidates`**, maps them to groups, applies **`combinations.json`** (same as automatic promotion), runs **`SyncGroupStageOutcomes`**, and responds with **200** and the same **`{ "data": … }`** shape as a successful match update.

```json
{
  "team_fifa_codes": ["ECU", "MEX", "BIH", "NOR", "SWE", "MAR", "TUR", "EGY"]
}
```

- **400** if the bracket is not in conflict anymore, codes are not eight distinct candidates, or the combination cannot be applied.

---

## High-level sequence (single or bulk update)

After a successful **`UpdateMatchesResult`** (single row wrapped in a slice, or bulk), `MatchService` always calls **`SyncGroupStageOutcomes`** (`internal/services/match_service.go`).

```md
  AdminHandler
        │
        │  UpdateMatchResult / UpdateMatchResultsBulk
        ▼
  MatchService
        │
        │  UpdateMatchesResult(updates)
        ▼
  MatchRepository ─────────────────────────► ok / error (stops here on error)
        │
        │  SyncGroupStageOutcomes
        ▼
  MatchService
        │
        ├──► GroupStandingService.RecalculateStandings
        │         │
        │         ├──► MatchRepository.GetMatches (finished group stage)
        │         └──► GroupStandingRepository.UpdateGroupStandings (each group)
        │
        ├──► promoteGroupWinners (MatchRepository.UpdateMatchTeams for 1st/2nd slots)
        │
        ├──► MatchRepository.IsGroupStageFinished (72 / 72 ?)
        │
        ├─── no ──► HTTP 200: is_group_stage_finished = false
        │            (promotion_outcome omitted)
        │
        └─── yes ─► promoteThirdPlaceTeams
                      │
                      ├── GetThirdPlaceGroups (12 teams, position 3, ranked)
                      │
                      ├── cutoff tie too large for remaining spots?
                      │        │
                      │        ├─ yes ──► promotion_outcome.status = "conflict"
                      │        │           + candidates only
                      │        │           (no UpdateMatchTeams for third slots)
                      │        │
                      │        └─ no ───► applyThirdPlaceAssignments
                      │                      │
                      │                      ├── findCombination (sorted top-8 groups)
                      │                      ├── build + UpdateMatchTeams (8 away slots)
                      │                      └── promotion_outcome.status = "completed"
                      │                          + assignments (match_id, away_team_fifa_code)
                      │
                      └──► HTTP 200: is_group_stage_finished = true
                            + promotion_outcome (either branch above)
```

**Important:** The match row update is **committed** in its own transaction inside `MatchRepository.UpdateMatchesResult`. If **`SyncGroupStageOutcomes`** fails afterward (for example standings recalculation error), the HTTP handler returns an error **but the match scores already remain saved**. The manual **`POST /api/admin/standings/recalculate`** endpoint exists to retry sync from current data.

`ResetMatchResult` follows the same pattern: reset in the repository, then `SyncGroupStageOutcomes`.

---

## Step 1 — Persist match result

**`MatchRepository.UpdateMatchesResult`** (`internal/repositories/match_repository.go`):

1. Verifies **all** requested match IDs exist; if any are missing → **`domain.ErrMatchesNotFound`** → handler maps to **404** (`handleServiceError` + `MatchesNotFoundError`).
2. Runs updates in a **single DB transaction** (all succeed or all roll back).
3. For each row: sets `home_score`, `away_score`, `status`, `winner_team_fifa_code`, `updated_at`.

Winner rule (SQL `CASE`):

- Home wins → `winner_team_fifa_code = home_team_fifa_code`
- Away wins → `winner_team_fifa_code = away_team_fifa_code`
- Draw (`home_score == away_score`) → `winner_team_fifa_code = NULL` (must still satisfy DB check that winner is one of the two teams when non-null).

---

## Step 2 — Recalculate all group standings

**`GroupStandingService.RecalculateStandings`** (`internal/services/group_standing_service.go`):

1. Loads **all finished** matches with `stage_code = group_stage` via `MatchRepository.GetMatches`.
2. Partitions matches by `group_code` and, for each letter **A–L**, recomputes standings in parallel (`sync.WaitGroup`).
3. For each group: builds points / goals from finished matches, applies **overall** sort (points, goal difference, goals for), then **head-to-head** subtrees for ties, assigns `position` 1–4, **`UPDATE group_standings`** by `fifa_code`.

Only **finished** group-stage matches contribute. Groups with no finished matches produce an empty recalculation slice for that group (no standing rows updated in that pass for that group’s loop—see `recalculateStandingsByGroup` with empty match list).

---

## Step 3 — Promote 1st and 2nd place into knockout slots

**`MatchService.promoteGroupWinners`** runs **after** standings are fresh.

Per group **A–L** (concurrently):

1. **`MatchRepository.IsGroupFinished(groupCode)`** — true only when that group has **6** finished matches (`COUNT … status = 'finished'` = 6). Until then, promotion for that group is skipped.
2. If finished: **`GetGroupStandings(ctx, []string{groupCode}, nil)`** reads current positions.
3. **`buildGroupPositionMatchUpdates`** walks **`domain.MatchSlotRules`** (`internal/domain/match.go`). For every rule whose **home** or **away** source is `SourceGroupPosition` with matching `GroupCode`, it sets **`home_team_fifa_code`** or **`away_team_fifa_code`** to the FIFA code of the team currently at that **position** in the standings (1 = winner, 2 = runner-up).

So **winner and runner-up** enter the bracket wherever `MatchSlotRules` references `group_position` with positions **1** and **2** for that group (round of 32 and downstream rules use `winner` / `loser` for later rounds—those are separate from this step).

Collected updates are **sorted by `match_id`** then applied in **`MatchRepository.UpdateMatchTeams`** (transaction, `COALESCE` so only provided side is overwritten).

---

## Step 4 — Is the entire group stage done?

**`MatchRepository.IsGroupStageFinished`**: counts finished matches with `stage_code = 'group_stage'`; must equal **`12 * 6 = 72`**.

- If **false**: `SyncGroupStageOutcomes` returns **`{ "is_group_stage_finished": false }`** (no third-place promotion block).
- If **true**: proceeds to **third-place promotion** and includes **`promotion_outcome`** in the response when applicable.

---

## Step 5 — Third-placed teams: ranking, cutoff ties, combinations

### 5.1 Load all third-placed teams with stats

**`GroupStandingRepository.GetThirdPlaceGroups`** selects every team with **`group_standings.position = 3`**, joined to `teams`, ordered by:

1. `points` DESC  
2. `goal_difference` DESC  
3. `goals_for` DESC  

So the slice is **best third** first, worst third last (12 rows for a full tournament).

### 5.2 Cutoff tie (“conflict”) vs clean top 8

**`MatchService.promoteThirdPlaceTeams`** (`internal/services/match_service.go`):

- **`thirdPlaceCutoffBounds`**: starting from the team at **index 7** (the 8th-best third), finds the inclusive index range of every team **tied on the same** points, goal difference, and goals for as that 8th team (extends up and down the list).
- **`availableSpots`**: `8 - cutoffStartIndex` (how many of the top 8 slots are still “unambiguous” above the tie).
- If **`len(cutoffGroup) > availableSpots`**, the bracket cannot be resolved automatically: promotion returns **`status: "conflict"`** with **`candidates`** only (teams still in contention, with `is_tied` for the ambiguous tail).

**No** `UpdateMatchTeams` is performed in the conflict path.

### 5.3 Clean path: top 8 + FIFA combination table

If there is **no** cutoff conflict:

1. Take **`teams[:8]`** (the eight best third-placed teams by the SQL ordering).
2. Build **`qualifyingGroups`** = their `group_code` values, **`sort.Strings`** (canonical key for lookup).
3. **`findCombination`** scans embedded **`combinations.json`** (`//go:embed internal/services/data/combinations.json`) for an entry whose **`qualifying_third_place_groups`** equals that sorted slice (`domain.ThirdPlaceCombination`, field `QualifyingGroups` JSON tag).
4. **`buildThirdPlaceMatchUpdates`** uses the combination’s **`assignments`** map (keys like **`"1A"`** = first place of group **A**, values like **`"3E"`** = third place of group **E**):
   - Resolves third-place **FIFA** code from the top-8 team list for each assigned group letter.
   - For each first-place key, finds the **round-of-32** match id in **`MatchSlotRules`** where **home** is `SourceGroupPosition` position **1** for that group (implementation assumes the first-place team is on **home** for these slots).
   - Emits **`MatchTeamUpdate`** setting **`AwayTeamFifaCode`** to the resolved third-place team.

Updates are sorted by **`match_id`** and written via **`UpdateMatchTeams`**.

Returned payload: **`promotion_outcome.status = "completed"`** and **`assignments`** (match id + away FIFA code applied).

**Data dependency:** The embedded JSON must contain a row for every sorted 8-tuple that can occur in production. If **`findCombination`** returns `nil`, the current code would dereference a nil combination when applying assignments—operationally, keep **`combinations.json`** complete; treat a missing row as a deployment/data bug.

---

## Conflict resolution — `POST /api/admin/standings/third-place/resolve`

When the sync response has **`promotion_outcome.status == "conflict"`**, an operator picks **which eight third-placed sides advance** by posting **`team_fifa_codes`** (see request body above).

**`MatchService.ResolveThirdPlaceConflict`**:

- Recomputes the cutoff conflict from current **`GetThirdPlaceGroups`**; if not in conflict → **400**.
- Requires every submitted code to appear in **`candidates`** for that conflict.
- Reuses **`applyThirdPlaceAssignments`** (combination lookup + **`UpdateMatchTeams`**), then **`SyncGroupStageOutcomes`** and returns that outcome (**200**).

---

## Sync-only admin endpoint

**`POST /api/admin/standings/recalculate`** calls **`MatchService.SyncGroupStageOutcomes`** with **no** prior match write. Use when:

- Standing rows and knockout placeholders need to be rebuilt from whatever is already in `matches` / `group_standings`, or  
- A previous call failed after scores were committed.

Response shape is the same **`SyncGroupStageOutcomes`** as match update endpoints.

---

## HTTP response shape (`SyncGroupStageOutcomes`)

Returned as JSON **`data`** (`httpx.Response`), type **`domain.SyncGroupStageOutcomes`** (`internal/domain/group_standing.go`):

| Field | Meaning |
| ----- | ------- |
| `is_group_stage_finished` | `true` iff 72/72 group-stage matches are `finished`. |
| `promotion_outcome` | Present when the group stage is finished. Omitted when `false` above. |
| `promotion_outcome.status` | `"completed"` or `"conflict"`. |
| `promotion_outcome.assignments` | On success: which Ro32 match away slot received which FIFA code. |
| `promotion_outcome.candidates` | On conflict: who might still qualify + tie flags. |

All admin endpoints that return this payload wrap it in **`{ "data": … }`**.

**Group stage not finished** — `promotion_outcome` is omitted:

```json
{
  "data": {
    "is_group_stage_finished": false
  }
}
```

**Third-place conflict** — manual resolution needed (`POST /api/admin/standings/third-place/resolve`):

```json
{
  "data": {
    "is_group_stage_finished": true,
    "promotion_outcome": {
      "status": "conflict",
      "candidates": [
        { "position": 1, "fifa_code": "ECU", "is_tied": false },
        { "position": 2, "fifa_code": "MEX", "is_tied": false },
        { "position": 3, "fifa_code": "BIH", "is_tied": false },
        { "position": 4, "fifa_code": "NOR", "is_tied": false },
        { "position": 5, "fifa_code": "SWE", "is_tied": false },
        { "position": 6, "fifa_code": "MAR", "is_tied": false },
        { "position": 7, "fifa_code": "TUR", "is_tied": false },
        { "position": 8, "fifa_code": "EGY", "is_tied": true },
        { "position": 9, "fifa_code": "ALG", "is_tied": true }
      ]
    }
  }
}
```

**Third-place completed** — eight away slots filled (shape only; real `assignments` has eight rows):

```json
{
  "data": {
    "is_group_stage_finished": true,
    "promotion_outcome": {
      "status": "completed",
      "assignments": [
        { "match_id": 79, "away_team_fifa_code": "MEX" },
        { "match_id": 85, "away_team_fifa_code": "BRA" }
      ]
    }
  }
}
```

---

## Error summary (admin match flows)

| Condition | HTTP |
| ----------- | ---- |
| Bad match id in path | **400** |
| Invalid body / validation | **400** |
| Not admin | **403** |
| Not authenticated | **401** |
| Unknown match id | **404** |
| Bad third-place resolution (not in conflict, wrong codes, etc.) | **400** |
| Server error | **500** |
