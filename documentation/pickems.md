# Pickems

This document describes the two prediction systems offered to authenticated users — **Pickem** (tournament-level predictions) and **Match score picks** (per-match exact-score predictions) — and how scoring flows from match results into per-board rankings.

---

## Overview

There are two independent prediction surfaces. Both target the same underlying tournament (FIFA WCP 2026: 48 teams, 12 groups A–L).

| System | Granularity | Lock timing | Storage |
| --- | --- | --- | --- |
| **Pickem** | Tournament-wide: group standings + best thirds + bracket | First group match kickoff | `user_group_picks`, `user_best_third_picks`, `user_bracket_picks` |
| **Match score picks** | Per match: exact home/away score | Each match's individual `kickoff_at` (or status change) | `user_match_score_picks` |

Both systems funnel into a single immutable audit log (`score_events`), then sum into per-user totals (`user_scores.pickem_points`, `user_scores.match_score_points`, `user_scores.total_points`) projected per board at read time.

---

## Tournament structure

- 12 groups (A–L), 4 teams each
- Top 2 of each group + 8 best thirds advance → 32 teams in Round of 32 (R32)
- Knockout chain: R32 (16 matches, IDs 73–88) → R16 (8, IDs 89–96) → QF (4, IDs 97–100) → SF (2, IDs 101–102) → 3rd place (1, ID 103) → Final (1, ID 104)
- The bracket slot map is the single source of truth in `internal/domain/match.go` (`MatchSlotRules`)

---

## User flow — Pickem

Users have **one** pickem set, shared across every board they belong to.

1. `GET /api/pickems` — fetch current state (returns empty arrays + `is_locked: false` if first time)
2. User fills group tables (drag-and-drop 4 teams to positions 1–4 per group)
3. User selects 8 of the 12 third-place teams as best-thirds
4. `PUT /api/pickems/groups` — saves group + best-third picks (drafts allowed: any subset of 12 groups, 0–8 best-thirds)
5. User fills bracket — picks the winner of each of the 32 knockout matches
6. `PUT /api/pickems/bracket` — saves bracket picks (strict: all 32 picks required, group + best-thirds must already be complete)
7. User can revisit and update either step until tournament lock
8. After first group match kickoff: pickem is locked, `is_locked: true`, all writes return `400 PICKEM_LOCKED`

### Drafts

Pickem submission supports drafts natively — there is **no separate draft endpoint**. Each `PUT /api/pickems/groups` is a full replace of the user's current group + best-third state. Frontend should send the full current state on every save (otherwise unsent groups are deleted).

`is_complete` is `true` only when:

- All 12 groups have 4 valid teams in positions 1–4
- 8 best thirds picked (each from a position-3 row in submitted picks)
- 32 bracket picks covering match IDs 73–104

Bracket save is **not** drafts-friendly: it requires all 32 picks at once and the underlying group state must already be complete.

---

## User flow — Match score picks

Per-match score predictions, independent of pickem completion.

- **Read** — picks are embedded inline in `GET /api/matches` as a `user_pick` field per match. The endpoint uses optional auth: anonymous callers get `user_pick: null` for every row; authenticated callers get their own pick (`{ "home_score": int, "away_score": int }`) or `null` if none.
- **Write** — `PUT /api/matches/{id}/pick` with body `{ "home_score": 2, "away_score": 1 }`. Returns 204. Upserts (no separate create/update path).
- **Lock condition** (per match): `now > match.kickoff_at OR match.status != 'scheduled'` — locked picks return `400 MATCH_PICK_LOCKED`.

Score values are bounded: `0 ≤ home_score, away_score ≤ 20`.

---

## Bracket projection

The user-facing bracket view (32 slots) is **computed at read time** — never persisted. The projection algorithm:

1. Build a `(group_code, position) → Team` lookup from the user's saved group picks
2. Sort the user's 8 best-third picks by their group code; assign them to the 8 R32 best-third slots (match IDs 74, 77, 79, 80, 81, 82, 85, 87) in ascending match-id order
3. For each knockout match ID (73 → 104), look up the slot rule in `MatchSlotRules` and resolve home/away from:
   - `SourceGroupPosition` → group lookup
   - `SourceBestThird` → best-third slot assignment
   - `SourceWinner` → user's bracket pick for the source match
   - `SourceLoser` → the team in the source SF slot the user did NOT pick (used only for match 103 — the third-place match)

Slots that can't be resolved (because earlier picks are missing) come back as `null` and are populated as the user fills more picks.

---

## Scoring

Scoring is a side-effect of admin match-result updates and runs asynchronously in a goroutine, so the admin response is not blocked.

### Scoring rules

| Source | Awarded when | Default points |
| --- | --- | --- |
| `group_standing_pick` | Group is fully finished AND user predicted exact position | `SCORING_GROUP_POSITION_EXACT` (3) |
| `group_standing_pick` | Group is fully finished, user predicted top-2 but wrong position | `SCORING_GROUP_QUALIFIES` (1) |
| `best_third_pick` | After third-place qualifiers resolved AND user picked a team that actually advanced | `SCORING_BEST_THIRD` (2) |
| `bracket_pick` | Knockout match finished AND user picked the actual winner (R32 → Final) | `SCORING_ROUND_OF_32` (4), `SCORING_ROUND_OF_16` (6), `SCORING_QUARTERFINALS` (8), `SCORING_SEMIFINALS` (12), `SCORING_THIRD_PLACE` (6), `SCORING_FINAL` (20) |
| `match_score_pick` | Match finished AND predicted exact score | `SCORING_MATCH_SCORE_EXACT` (5) |
| `match_score_pick` | Match finished AND predicted correct outcome (winner/draw) but wrong score | `SCORING_MATCH_SCORE_OUTCOME` (2) |

All point values are env-configurable.

### Scoring trigger flow

```
Admin: POST /api/admin/matches/{id}/result        (or the bulk variant)
  → match_service.UpdateMatchesResult
    1. Update matches row(s)                       (sync, in tx)
    2. SyncGroupStageOutcomes:                     (sync)
       - RecalculateStandings
       - promote group winners (top-2 → R32)
       - if all 12 groups finished: promoteThirdPlaceTeams
    3. fire goroutine: ScoreMatches(matchIDs)      (async)
       - if step 2 auto-resolved third-place: also fire ScoreBestThirds

Admin: POST /api/admin/standings/third-place/resolve  (manual conflict resolve)
  → match_service.ResolveThirdPlaceConflict
    1. apply assignments (sync)
    2. SyncGroupStageOutcomes (sync)
    3. fire goroutine: ScoreBestThirds (async)
```

`ScoreMatches([]int64)` is batched: it filters to finished matches at the DB layer, dedups groups (if multiple matches in the same group are in the batch, group_standing scoring runs once for that group), batches all `score_events` upserts into one query, and updates all affected `user_scores` rows in one query.

`ScoreBestThirds` derives the 8 actually-advancing thirds from the populated R32 slots (match IDs 74, 77, 79, 80, 81, 82, 85, 87) — all of which have their `away_team_fifa_code` set by the third-place resolution flow — and awards points to every user who picked any of those 8 teams.

### `score_events` audit log

| `source_type` | `source_ref` format | Example | Reads as |
| --- | --- | --- | --- |
| `group_standing_pick` | `<group>:<team_fifa_code>` | `A:MEX` | "Mexico's predicted position in Group A was correct" |
| `best_third_pick` | `<team_fifa_code>` | `MEX` | "Mexico correctly picked to advance as a best third" |
| `bracket_pick` | `<match_id>` | `73` | "Picked the winner of match 73 (R32)" |
| `match_score_pick` | `<match_id>` | `17` | "Predicted the exact score (or correct outcome) of match 17" |

Idempotent: the table has `UNIQUE (user_id, source_type, source_ref)` and inserts use `ON CONFLICT DO UPDATE SET points`. Re-running scoring for the same match (after a result correction, for example) updates existing rows in place; it never duplicates.

### User scores propagation

After every batched score_events upsert, `user_scores` is recomputed (in a single query) for the union of affected user IDs:

- `pickem_points       = SUM(points) WHERE source_type IN (group_standing_pick, best_third_pick, bracket_pick)`
- `match_score_points  = SUM(points) WHERE source_type = match_score_pick`
- `total_points        = pickem_points + match_score_points`
- `exact_hits          = COUNT(*) WHERE source_type = match_score_pick AND points >= MatchScoreExact`
- `correct_outcomes    = COUNT(*) WHERE source_type = match_score_pick AND points > 0`

`user_scores` is keyed by `user_id` only — the rank within a specific board is derived at read time via a `RANK() OVER (...)` window function joined with `board_members`. Score events and per-user totals are board-agnostic; only the rank projection is per-board.

### Joining a board

`POST /api/boards/join` is a pure `board_members` insert. No backfill is needed: `user_scores` rows are seeded at signup (in the same transaction as the user row), so the leaderboard JOIN finds the user from day one.

---

## Tables

| Table | Purpose |
| --- | --- |
| `user_group_picks` | Per-user predicted position (1–4) for each team in each group |
| `user_best_third_picks` | Per-user picks: which 8 teams advance as best thirds |
| `user_bracket_picks` | Per-user predicted winner for each knockout match |
| `user_match_score_picks` | Per-user exact-score prediction per match |
| `score_events` | Append-only audit of all awarded points (idempotent on `(user_id, source_type, source_ref)`) |
| `user_scores` | Per-user aggregate totals — recomputed from `score_events` after each scoring run; PK on `user_id` only (board-agnostic, see `cmd/db/migrations/000006_create_user_scores_table.up.sql`) |

See `cmd/db/migrations/000013_create_pickem_tables.up.sql` for the picks/score-events schema.

---

## Observability and admin operations

Scoring runs as a fire-and-forget goroutine. Both success and failure paths emit a single structured log line (`scope`, `match_ids` / `match_id`, `users_affected`, and `error` on failure). There is no audit table — logs are the source of truth for "did this scoring run succeed?".

Retry path: re-post the match result via `POST /api/admin/matches/{id}/result`. Scoring is fully idempotent — `score_events` upsert on `(user_id, source_type, source_ref)`, and `user_scores` is recomputed from scratch via `BatchUpdateUserScores` — so re-running converges to the correct state regardless of how many times.

Admin tools:

| Method | Path | Use |
| --- | --- | --- |
| `POST` | `/api/admin/pickems/rescore/match/{id}` | Manually re-run scoring for a single match (idempotent) |
| `POST` | `/api/admin/pickems/rescore/best-thirds` | Manually re-run best-thirds scoring (idempotent; returns `400 BEST_THIRDS_NOT_SCOREABLE` if third-place qualifiers haven't been resolved) |

All rescore endpoints are safe to call repeatedly — the upsert semantics on `score_events` and the recompute-from-events `BatchUpdateUserScores` make scoring fully idempotent.

---

## Glossary

- **Pickem** — the tournament-level prediction system (group standings + best thirds + bracket).
- **Match score pick** — a per-match exact home/away score prediction.
- **Best third** — a team that finishes 3rd in its group AND qualifies among the top-8 third-placed teams across all 12 groups (per FIFA tiebreakers).
- **Bracket pick** — the user's predicted winner of a single knockout match.
- **`source_ref`** — the per-event identifier within a `source_type`. Combined with `(user_id, source_type)` it forms a unique key on `score_events`.
- **Lock** — pickem locks at first group match kickoff; match score picks lock per match (kickoff or status transition).
- **Idempotent rescore** — re-running scoring overwrites existing event rows in place and recomputes board totals from the full event log; counts and totals converge to the same final state regardless of how many times scoring runs.
