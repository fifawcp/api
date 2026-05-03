# Private Board Management

Private groups of friends with join codes, member roles, and internal rankings. All picks and scores are scoped to boards.

---

## Endpoints

| Method | Endpoint | Auth | Purpose |
| ------ | -------- | ---- | ------- |
| POST | `/api/boards` | Bearer | Create a new board (user becomes owner) |
| GET | `/api/boards` | Bearer | List all boards the authenticated user belongs to |
| GET | `/api/boards/{boardId}` | Bearer + Member | Get board details |
| POST | `/api/boards/join` | Bearer | Join a board using a join code |
| GET | `/api/boards/{boardId}/members` | Bearer + Member | List all members of a board |
| GET | `/api/boards/{boardId}/ranking` | Bearer + Member | Get the internal ranking for a board |
| POST | `/api/boards/{boardId}/regenerate-join-code` | Bearer + Member + Admin | Regenerate the join code |
| PATCH | `/api/boards/{boardId}` | Bearer + Member + Admin | Update board name |
| DELETE | `/api/boards/{boardId}` | Bearer + Member + Owner | Delete board |
| PATCH | `/api/boards/{boardId}/members/{userId}/role` | Bearer + Member + Admin | Update member role |
| DELETE | `/api/boards/{boardId}/members/{userId}` | Bearer + Member + Admin | Remove member |

---

## Flows

### Create Board

```md
POST /api/boards
Request:  { name: "My Board" }
Response: {
  id: "uuid",
  name: "My Board",
  owner_user_id: "uuid",
  join_code: "ABCD1234",
  created_at: "2026-01-01T00:00:00Z"
}
Action:
  - Validate user is authenticated (via Auth middleware)
  - Generate 8-character uppercase alphanumeric join code
  - Retry on unique constraint violation (join_code collision)
  - Atomic CTE: INSERT board → INSERT board_members (owner, admin) → INSERT board_rankings
  - Return created board with ID and created_at
```

---

### Join Board

```md
POST /api/boards/join
Request:  { join_code: "ABCD1234" }
Response: 204 No Content
Action:
  - Validate user is authenticated
  - Atomic CTE: Lookup board by join_code → INSERT board_members (user, member) → INSERT board_rankings
  - If join_code invalid → 401 "invalid or expired board join code"
  - If user already member → 409 "user is already a member of this board"
  - Board ranking cascade deleted via FK when member removed
```

---

### Get Board Details

```md
GET /api/boards/{boardId}
Response: {
  id: "uuid",
  name: "My Board",
  owner_user_id: "uuid",
  join_code: "ABCD1234",
  created_at: "2026-01-01T00:00:00Z"
}
Action:
  - Validate UUID format (400 if invalid)
  - Check board exists (404 if not found)
  - Check user is board member (403 if not)
  - Inject board_id and user_role into request context
  - Return board details
```

---

### List Board Members

```md
GET /api/boards/{boardId}/members
Response: [
  {
    board_id: "uuid",
    user_id: "uuid",
    role: "admin" | "member",
    username: "john_doe",
    created_at: "2026-01-01T00:00:00Z"
  }
]
Action:
  - RequireBoardMembership middleware validates access
  - JOIN board_members with users to include username
  - ORDER BY created_at DESC
  - Return empty array [] if no members
```

---

### Get Board Ranking

```md
GET /api/boards/{boardId}/ranking
Response: [
  {
    user_id: "uuid",
    rank: 1,
    total_points: 100,
    pickem_points: 50,
    match_score_points: 50,
    exact_hits: 5,
    correct_outcomes: 10,
    updated_at: "2026-01-01T00:00:00Z"
  }
]
Action:
  - RequireBoardMembership middleware validates access
  - SELECT from board_rankings WHERE board_id = ?
  - ORDER BY total_points DESC
  - Frontend joins with members endpoint by user_id to get usernames
```

---

### Regenerate Join Code

```md
POST /api/boards/{boardId}/regenerate-join-code
Response: { join_code: "EFGH5678" }
Action:
  - RequireBoardMembership middleware validates access
  - Service checks user_role from context
  - If not admin → 403 "insufficient permissions"
  - Generate new 8-character join code
  - UPDATE boards SET join_code = ?
  - Return new join code
```

---

### Update Board

```md
PATCH /api/boards/{boardId}
Request:  { name: "Updated Board Name" }
Response: 204 No Content
Action:
  - RequireBoardMembership middleware validates access
  - Service checks user_role from context
  - If not admin → 403 "insufficient permissions"
  - Dynamic UPDATE query: only update non-zero fields
  - Currently supports: name
  - Return 204 No Content on success
```

---

### Delete Board

```md
DELETE /api/boards/{boardId}
Response: 204 No Content
Action:
  - RequireBoardMembership middleware validates access
  - DELETE FROM boards WHERE id = ? AND owner_user_id = ?
  - Check rows affected
  - If 0 rows affected
    → 403 "insufficient permissions" (not owner)
    → 404 "board not found"
  - Cascade deletes board_members and board_rankings via FK
  - Return 204 No Content on success
```

---

### Update Member Role

```md
PATCH /api/boards/{boardId}/members/{userId}/role
Request:  { role: "admin" | "member" }
Response: 204 No Content
Action:
  - RequireBoardMembership middleware validates access
  - Service checks user_role from context
  - If not admin → 403 "insufficient permissions"
  - Check if target user is board owner via subquery
  - If target is owner → 403 "insufficient permissions"
  - UPDATE board_members SET role = ? WHERE board_id = ? AND user_id = ?
  - Check rows affected
  - If 0 rows affected
    → 403 "insufficient permissions" (try to update owner's role)
    → 404 "board member not found"
  - Return 204 No Content on success
```

---

### Remove Member

```md
DELETE /api/boards/{boardId}/members/{userId}
Response: 204 No Content
Action:
  - RequireBoardMembership middleware validates access
  - Service checks user_role from context
  - If not admin → 403 "insufficient permissions"
  - Check if target user is board owner via subquery
  - If target is owner → 403 "insufficient permissions"
  - DELETE FROM board_members WHERE board_id = ? AND user_id = ?
  - Board ranking cascade deleted via FK
  - Check rows affected
  - If 0 rows affected
    → 404 "board member not found"
    → 403 "insufficient permissions" (try to delete owner)
  - Return 204 No Content on success
```

---

## Business Rules

- Only authenticated users can create or join boards
- A user cannot join a board they already belong to
- Only board admins (including owner) can regenerate the join code
- All picks and scores are scoped to `(board_id, user_id)` pairs
- A user can belong to multiple boards with independent picks per board
- Board owner cannot have their role changed or be removed from the board
- Backend is the source of truth for membership validation
- Frontend joins members and ranking data by user_id for display

---

## Database Schema

### boards

| Column | Type | Notes |
| ------ | ---- | ----- |
| id | UUID | Primary key |
| name | varchar(120) | Not null |
| owner_user_id | UUID | FK → users |
| join_code | varchar(8) | Unique, not null |
| created_at | timestamp(0) with time zone | |

### board_members

| Column | Type | Notes |
| ------ | ---- | ----- |
| board_id | UUID | FK → boards, PK |
| user_id | UUID | FK → users, PK |
| role | varchar(20) | 'admin' or 'member', default 'member' |
| created_at | timestamp(0) with time zone | |

### user_scores

Per-user totals — one row per user, board-agnostic. Per-board rank is computed at read time via a `RANK()` window function joined with `board_members`.

| Column | Type | Notes |
| ------ | ---- | ----- |
| user_id | UUID | FK → users, PK |
| total_points | int | Default 0; sum of pickem_points and match_score_points (and future sources) |
| pickem_points | int | Default 0; points from bracket / group / best-third pickems (see `pickems.md`) |
| match_score_points | int | Default 0; points from match score picks (see `pickems.md`) |
| exact_hits | int | Default 0 |
| correct_outcomes | int | Default 0 |
| updated_at | timestamp(0) with time zone | |

**Lifecycle:** the row is created at user signup (in the same transaction as the `users` INSERT) and updated by scoring runs.

---

## Middleware & Access Control

### Auth Middleware

- Applied to `/boards/*` routes
- Extracts authenticated user from JWT
- Sets user in request context

### RequireBoardMembership Middleware

- Applied to `/boards/{boardId}/*` routes
- Validates board_id UUID format (400 if invalid)
- Checks board exists (404 if not found)
- Checks user is board member (403 if not)
- Injects board_id and user_role into request context
- Single middleware call for all board-specific routes

## RequireValidUserIdMiddleware

- Applied to `/boards/{boardId}/members/{userId}/*` routes
- Validates userId UUID format (400 if invalid)
- Single middleware call for all member-specific routes

### Role-Based Access

- Service layer checks user_role from context for admin-only actions
- Returns 403 "insufficient permissions" for non-admin users
- Used for: regenerate join code, update board, delete board, manage members

---

## Error Reference

| Scenario | Status | Message |
| -------- | ------ | ------- |
| Invalid board ID format | 400 | `invalid board ID` |
| Board not found | 404 | `board not found` |
| User not a board member | 403 | `not a member of this board` |
| Invalid join code | 401 | `invalid or expired board join code` |
| User already in board | 409 | `user is already a member of this board` |
| Insufficient permissions (non-admin) | 403 | `insufficient permissions` |
| Board owner cannot be modified | 403 | `insufficient permissions` |
| Board member not found | 404 | `board member not found` |
| Internal server error | 500 | `internal server error` |
