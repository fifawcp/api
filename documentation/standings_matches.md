# World Cup read-only data (standings / matches)

Public, unauthenticated HTTP endpoints that expose seeded tournament data: group standings (with teams) and the match schedule with optional filters.

---

## Endpoints

| Method | Endpoint | Auth | Purpose |
| ------ | -------- | ---- | ------- |
| GET | `/api/standings` | None | List group standings rows (optionally filtered by group and/or table position) |
| GET | `/api/matches` | None | List matches with optional filters, ordered by kickoff |

---

## GET `/api/standings`

Returns a **flat** list of `group_standings` rows joined to `teams`, ordered by `group_code` then `position`. The client can partition rows by `team.GroupCode` if it needs a nested “groups” UI.

### Query parameters

| Parameter | Type | Required | Description |
| --------- | ---- | -------- | ----------- |
| `group_codes` | string list | No | Restrict to one or more groups (`A`–`L`). Case-insensitive; normalized to uppercase. |
| `position` | int64 | No | Restrict to a single table rank (e.g. `3` for all third-placed teams across the filtered groups). |

**List encoding:** For query parameter `group_codes`, values can be repeated (`group_codes=A&group_codes=B`) and/or comma-separated (`group_codes=A,B`) **Preferred**.

**Validation:** Each entry in `group_codes` must be a valid group letter after uppercasing; otherwise **400** with `invalid group code`. If `position` is present but not a valid integer, **400** with an invalid-parameter message.

### Example response (`200`)

```json
{
  "data": [
    {
      "position": 1,
      "team": {
        "fifa_code": "MEX",
        "name": "Mexico",
        "flag_url": "https://…",
        "group_code": "A"
      },
      "matches_played": 0,
      "wins": 0,
      "draws": 0,
      "losses": 0,
      "goals_for": 0,
      "goals_against": 0,
      "goal_difference": 0,
      "points": 0
    }
  ]
}
```

---

## GET `/api/matches`

Returns matches with home and away team rows joined from `teams`, **`ORDER BY m.kickoff_at ASC`**.

### Query parameters

| Parameter | Type | Required | Description |
| --------- | ---- | -------- | ----------- |
| `group_codes` | string list | No | Filter by group code. Each value must be `A`–`L`. |
| `stage_code` | string list | No | Filter by stage code (see **Stage codes and validation** below). |
| `status` | string | No | Filter by match status: `scheduled` or `finished`. |
| `team_fifa_codes` | string list | No | Filter matches where the team appears as home or away. |
| `from_date` | string | No | Return matches on or after this kickoff date-time. |
| `to_date` | string | No | Return matches on or before this kickoff date-time. |

**List encoding:** Same repeat-or-comma rules as `group_codes` for all list parameters (`httpx.ParseStringSliceParam`).

**Date format:** Both dates are parsed with **`time.RFC3339`** (e.g. `2026-06-15T00:00:00Z`). Omit the parameter entirely if you do not want that bound. Malformed values → **400** (`invalid 'from_date' date format, expected RFC3339` or the same for `to_date`).

**Date range rule:** If both `from_date` and `to_date` are set, `from_date` must be before or equal to `to_date` (`validator.IsValidDateRange`); otherwise **400** (`invalid date range`).

**Filter combination:** Every populated filter category is combined with **AND** in SQL. Within a category:

- Multiple `group_code` values → `IN` (match in any listed group).
- Multiple `stage_code` values → `IN`.
- Multiple `team_fifa_code` values → `(home = t1 OR away = t1) OR (home = t2 OR away = t2) OR …`.

### Stage codes and validation

**Stored on rows / returned in JSON** (`domain.MatchStageCode`, aligned with the DB `CHECK`):  
`group_stage`, `round_of_32`, `round_of_16`, `quarterfinals`, `semifinals`, `third_place`, `final`.

**Accepted on the query string today** (`internal/infrastructure/validator/validators.go`):  
`group_stage`, `round_of_16`, `quarter_finals`, `semi_finals`, `third_place`, `final`.

So the filter whitelist and the database literals are **not identical** (naming and missing `round_of_32`). Clients should treat query validation as authoritative for “what the API accepts until the validator is aligned,” and treat response bodies as authoritative for “what is stored.”

### Example requests

```md
GET /api/matches
GET /api/matches?group_code=A
GET /api/matches?stage_code=group_stage&status=finished
GET /api/matches?team_fifa_codes=BRA,MEX
GET /api/matches?from_date=2026-06-15T00:00:00Z&to_date=2026-06-20T23:59:59Z
```

### Example match object (`200`)

```json
{
  "data": [
    {
      "id": 1,
      "stage_code": "group_stage",
      "group_code": "A",
      "home_team": {
        "FifaCode": "MEX",
        "Name": "Mexico",
        "FlagURL": "https://…",
        "GroupCode": "A"
      },
      "away_team": {
        "FifaCode": "RSA",
        "Name": "South Africa",
        "FlagURL": "https://…",
        "GroupCode": "A"
      },
      "kickoff_at": "2026-06-11T14:00:00Z",
      "status": "scheduled",
      "home_score": null,
      "away_score": null,
      "winner_team_fifa_code": null,
      "updated_at": "2026-06-11T12:00:00Z"
    }
  ]
}
```

Empty result sets return `"data": []`.
