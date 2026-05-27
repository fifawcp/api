# Best third-place promotion (reference)

Implementation: **`internal/services/match_service.go`**. Data: **`internal/services/data/combinations.json`** (embedded at compile time). Slot wiring: **`domain.MatchSlotRules`** in **`internal/domain/match.go`**.

## When it runs

Only from **`MatchService.SyncGroupStageOutcomes`** after:

1. Standings recalculation and group-winner promotion have run.  
2. **`MatchRepository.IsGroupStageFinished`** is **true** (all `group_stage` matches finished).

If the group stage is not finished, the sync response omits third-place work (`is_group_stage_finished: false`).

## Inputs

**`GroupStandingRepository.GetThirdPlaceGroups`** — all teams with **`group_standings.position = 3`**, ordered by **`points DESC`**, **`goal_difference DESC`**, **`goals_for DESC`**, joined to **`teams`** for FIFA and group codes (see `internal/repositories/group_standing_repository.go`).

Tiebreakers for this ordering match the third-place mini-table only as far as **points, goal difference, goals scored** (no fair play / lots).

The promotion logic assumes **at least eight** rows (eighth index used for the cutoff). A full tournament yields twelve.

## Combination system (`combinations.json`)

### Why it exists

There are **twelve** groups, so **twelve** third-placed teams after the group stage. **Eight** of them continue into the round of 32 as “best thirds.” FIFA’s fixed bracket (which first-place side plays a third, and which allowed third **group letters** can meet which host) means you cannot treat advancement as “any eight groups out of twelve”: only certain **sets of eight group letters** are legal together.

The embedded file lists those cases: **495** rows (FIFA’s enumeration). Each row is one globally consistent outcome: “these eight groups’ thirds advance, and here is exactly how each is slotted against the eight Ro32 fixtures that take a best-third away team.”

### What one row contains

Typical JSON shape (extra fields are ignored by the Go loader but stay in the file for humans / tooling):

```json
{
  "option": 1,
  "qualifying_third_place_groups": ["E", "F", "G", "H", "I", "J", "K", "L"],
  "eliminated_third_place_groups": ["A", "B", "C", "D"],
  "assignments": {
    "1A": "3E",
    "1B": "3J",
    "1D": "3I",
    "1E": "3F",
    "1G": "3H",
    "1I": "3G",
    "1K": "3L",
    "1L": "3K"
  }
}
```

- **`qualifying_third_place_groups`** — the **eight** group letters whose third-placed teams advance in this scenario (order in the array is **not** used for lookup; see below).  
- **`eliminated_third_place_groups`** — the other four groups’ thirds are out (present in the JSON; **not** unmarshalled into **`domain.ThirdPlaceCombination`** today).  
- **`assignments`** — eight entries. Key **`"1A"`** means “the Ro32 match where **first place in group A** is the seeded group winner for that slot.” Value **`"3E"`** means “the **third-placed team from group E** fills the **best-third** away side of that same match.” So the map is not a pool: it is the **exact** E→that slot pairing for this qualifying set.

Other first-place slots (e.g. **1C**, **1F**, …) play a **second**-placed team in this tournament design; they do not appear in this `assignments` map.

### How the code picks a row

1. From standings, take the **eight best** thirds and their **`group_code`** letters.  
2. **`sort.Strings`** on those eight letters so the set has a **canonical** key (same multiset, any order from SQL → same slice).  
3. **`findCombination`** scans the embedded slice until **`slices.Equal(combo.QualifyingGroups, thatSortedSlice)`** — **`QualifyingGroups`** maps JSON **`qualifying_third_place_groups`**.  
4. **`combo.Assignments`** drives **`buildThirdPlaceMatchUpdates`**: for each pair, resolve the third’s **FIFA** code from the eight loaded **`ThirdPlaceTeam`** rows and set the correct Ro32 **`away_team_fifa_code`** via **`MatchSlotRules`** (see Path A).

If the sorted top-eight letters do not match **any** row, the combination is missing from the dataset (deployment / data bug for a real tournament).

## Outcomes (`PromoteThirdPlaceTeams`)

| `status` | Meaning |
| -------- | ------- |
| **`completed`** | Top eight are unambiguous; Ro32 **away** “best third” slots were updated; **`assignments`** lists `match_id` + `away_team_fifa_code`. |
| **`conflict`** | Eighth-ninth (and neighbours) cannot be split on implemented stats; **`candidates`** lists who may still qualify; **no** `UpdateMatchTeams` for third slots in this pass. |

## Path A — no cutoff conflict

1. Take **`teams[:8]`** from the ordered list.  
2. Build **`qualifyingGroups`** = their **`group_code`** values, **`sort.Strings`** (canonical key).  
3. **`findCombination`** — linear scan of embedded **`[]ThirdPlaceCombination`** until **`slices.Equal(combo.QualifyingGroups, qualifyingGroups)`**.  
4. **`buildThirdPlaceMatchUpdates`** — for each JSON assignment entry **`"1A" → "3E"`** (first-place group **A** hosts; third from group **E** on the best-third side):
   - Parse group letters from the two runes after **`1`** and **`3`**.  
   - Resolve the third’s **FIFA** code from the eight **`ThirdPlaceTeam`** rows.  
   - Find **`MatchSlotRules`** entry whose **home** is **`SourceGroupPosition`**, position **1**, matching that first-place group (implementation assumes first place is **home** for these Ro32 rules).  
   - Emit **`MatchTeamUpdate`** with **`AwayTeamFifaCode`** only.  
5. Sort updates by **`match_id`**, **`UpdateMatchTeams`**, return **`assignments`** for the API payload.

**`SourceBestThird`** on the rule is only structural; **`GroupCodes`** on that source are not used for picking teams — the JSON map decides which third goes where.

## Path B — cutoff conflict

**`evaluateThirdPlaceCutoffConflict`** uses **`thirdPlaceCutoffBounds`**: every team tied with list index **7** (the eighth best third) on **points, goal difference, goals for** (contiguous block in the sorted list).

Let **`cutoffLen`** be that block’s size and **`cutoffStart`** its first index. There are **`8 - cutoffStart`** advancing slots still available for everyone from **`cutoffStart`** upward **within the top eight list positions**. If **`cutoffLen > 8 - cutoffStart`**, too many sides share the same stats for those slots → **conflict**.

**`thirdPlaceCandidates`** returns the prefix of the full list through the tied block: guaranteed qualifiers plus the ambiguous band, with **`is_tied: true`** from **`cutoffStart`** onward in that slice.

## Manual resolution

**`POST /api/admin/standings/third-place/resolve`** — body **`team_fifa_codes`**: exactly **eight** distinct codes (payload normalized with trim + uppercase).

1. Reload **`GetThirdPlaceGroups`**, re-run conflict detection; if not in conflict → **`ErrThirdPlaceNotInConflict`**.  
2. Every code must appear in the current **`candidates`** list.  
3. Build **`[]*ThirdPlaceTeam`** in payload order, **`applyThirdPlaceAssignments`** (same as automatic path), then full **`SyncGroupStageOutcomes`**.

If no JSON combination exists for the chosen eight groups, **`applyThirdPlaceAssignments`** fails (see domain errors / handler mapping).

**`domain.ThirdPlaceCombination`** only unmarshals **`qualifying_third_place_groups`** → **`QualifyingGroups`** and **`assignments`** → **`Assignments`**; **`option`** and **`eliminated_third_place_groups`** remain in the file for reference only.

## Not implemented

Fair play and drawing of lots for cross-group third-place ordering; richer FIFA tiebreaks beyond points / GD / GF for the **global** third-place table.

## Related

End-to-end HTTP and sync ordering: **`documentation/match_result_update_flow.md`**.
