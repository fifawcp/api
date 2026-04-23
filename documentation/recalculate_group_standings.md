# Group standings calculation (reference)

Implementation: `internal/services/group_standing_service.go`. Entry point: **`GroupStandingService.RecalculateStandings`** — loads all **finished** **`group_stage`** matches, buckets them by `group_code`, and runs **`recalculateStandingsByGroup`** for groups **A–L** in parallel. Persisted with **`GroupStandingRepository.UpdateGroupStandings`**.

## Inputs

- Only matches with **`status = finished`** and **`stage_code = group_stage`** (see `GetMatches` filters in `RecalculateStandings`).

## Per-row statistics (overall group)

For each team, over **all** group matches, the code accumulates:

| Field | Rule |
| ----- | ---- |
| `matches_played` | +1 per match |
| `wins` / `draws` / `losses` | Win 3 pts, draw 1 each, loss 0 |
| `points` | 3 win, 1 draw |
| `goals_for` / `goals_against` | Sum goals scored and conceded |
| `goal_difference` | `goals_for - goals_against` |

(`calculateOverallStats`)

## Ordering (FIFA-style, partial)

### 1. First ordering — overall mini-table

Sort all four teams by **`sortByOverallStats`**:

1. **Points** (higher first)  
2. **Goal difference** (higher first)  
3. **Goals for** (higher first)

### 2. Ties on (points + GD + GF)

**`identifyTiedGroups`** finds **consecutive** blocks in that sorted list where **all three** match between neighbours. Each block with **two or more** teams is a tie group.

### 3. Head-to-head within each tie group

For each tie group:

1. **`filterHeadToHeadMatches`** — matches where **both** sides are in the tie group.  
2. **`calculateHeadToHeadStats`** — same W/D/L and goal rules as overall, but only on those matches (teams with no H2H games get zeroed mini-rows from the map lookup in sort).  
3. **`sortTiedGroupByHeadToHead`** — sort the subgroup by:  
   - head-to-head **points**  
   - then head-to-head **goal difference**  
   - then head-to-head **goals for**

### 4. Still tied on head-to-head?

If **`isStillTied`** (everyone in the subgroup still equal on H2H points, H2H GD, H2H GF), **`sortByOverallStats`** is applied **again** to that subgroup only (code comment: FIFA rules “d, e” style fallback).

### 5. Positions

After tie groups are patched back into the main `standings` slice (same indices as after step 1), **`position`** is set to **`index + 1`** in slice order.

## Not implemented

Fair play, drawing of lots, and any tiebreak beyond the steps above.

## When this runs

- **`MatchService`** calls **`RecalculateStandings`** after group-stage match writes/resets/bulk sync (see `documentation/match_result_update_flow.md`).
- Admin **`POST …/standings/recalculate`** triggers the same service method.
