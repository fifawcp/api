# Passwordless Authentication

Email-based OTP with JWT access tokens, rotatable refresh tokens, and multi-device session management.

---

## Endpoints

| Method | Endpoint | Auth | Purpose |
| ------ | -------- | ---- | ------- |
| POST | `/api/auth/otp/request` | None | Request OTP (registration or login) |
| POST | `/api/auth/token` | None | Exchange OTP for tokens |
| POST | `/api/auth/token/refresh` | Cookie | Rotate refresh token + get new access token |
| POST | `/api/auth/logout` | Cookie | Logout current device |
| POST | `/api/auth/logout/all` | Cookie | Logout all devices |
| GET | `/api/auth/sessions` | Cookie | List active sessions |
| DELETE | `/api/auth/sessions/{id}` | Bearer | Delete a specific session |
| GET | `/api/users/profile` | Bearer | Get authenticated user profile |

---

## Flows

### Registration

```md
1. POST /auth/otp/request
   Request:  { identifier: "user@email.com", purpose: "registration" }
   Response: { message: "OTP sent" }
   Action:
     - Check user does NOT exist (returns 401 if already registered)
     - Generate 6-digit OTP
     - Store OTP hash in Redis (10min TTL, max 3 attempts)
     - Send OTP via email (TODO)

2. POST /api/auth/token
   Request: {
     identifier: "user@email.com",
     otp: "123456",
     purpose: "registration",
     user: { username, first_name, last_name }
   }
   Response (JSON): {
     access_token: "jwt...",
     expires_at: "2026-01-01T00:15:00Z",
     user: { id, email, username, first_name, last_name }
   }
   Response (Cookie): Set-Cookie: refresh_token=rt...; HttpOnly; Secure; SameSite=None; Path=/api/auth
   Action:
     - Verify OTP hash (401 if wrong or expired, 429 if too many attempts)
     - Create user in DB
     - Create session (device info, IP address, user agent)
     - Generate access token (15min JWT)
     - Generate refresh token (7-day random token, stored as SHA256 hash in DB)
     - Set refresh token as HttpOnly cookie (7 days)
     - Delete OTP from Redis
```

---

### Login

```md
1. POST /auth/otp/request
   Request:  { identifier: "user@email.com", purpose: "login" }
   Response: { message: "OTP sent" }
   Action:
     - Check user EXISTS (returns 401 if not found)
     - Generate 6-digit OTP
     - Store OTP hash in Redis (10min TTL, max 3 attempts)
     - Send OTP via email (TODO)

2. POST /api/auth/token
   Request: {
     identifier: "user@email.com",
     otp: "123456",
     purpose: "login"
   }
   Response (JSON): {
     access_token: "jwt...",
     expires_at: "2026-01-01T00:15:00Z",
     user: { id, email, username, first_name, last_name }
   }
   Response (Cookie): Set-Cookie: refresh_token=rt...; HttpOnly; Secure; SameSite=None; Path=/api/auth
   Action:
     - Verify OTP hash (401 if wrong or expired, 429 if too many attempts)
     - Fetch existing user
     - Create session (device info, IP address, user agent)
     - Generate access token (15min JWT)
     - Generate refresh token (7-day random token, stored as SHA256 hash in DB)
     - Set refresh token as HttpOnly cookie (7 days)
     - Delete OTP from Redis
```

---

### Token Refresh

```md
POST /api/auth/token/refresh
Request:  (no body — refresh token read from cookie automatically)
Response (JSON): {
  access_token: "jwt...",
  expires_at: "2026-01-08T00:00:00Z"
}
Response (Cookie): Set-Cookie: refresh_token=rt_new...; HttpOnly; Secure; SameSite=None; Path=/api/auth
Action:
  - Read refresh token from HttpOnly cookie (401 if missing)
  - Lookup token hash in DB (must exist + not expired → 401 otherwise)
  - Atomic CTE: DELETE old token + INSERT new token in one query
  - Update session last_used_at
  - Set new refresh token cookie (7 days)
  - Return new access token (15min)

Notes:
  - Both tokens are always rotated together
  - If a previously rotated token is replayed → lookup fails → 401
  - Old token is invalidated atomically — no window for race conditions
```

---

### Logout (current device)

```md
POST /api/auth/logout
Request:  (no body — refresh token read from cookie automatically)
Response: 204 No Content + clears the refresh_token cookie
Action:
  - Read refresh token from HttpOnly cookie (401 if missing)
  - Hash the incoming refresh token
  - DELETE session WHERE id = (SELECT session_id FROM refresh_tokens WHERE token_hash = ?)
  - Refresh token is cascade deleted via FK
  - Clear the refresh_token cookie

Notes:
  - If token not found or expired → 401
  - Only affects the session associated with this refresh token
```

---

### Logout All (all devices)

```md
POST /api/auth/logout/all
Request:  (no body — refresh token read from cookie automatically)
Response: 204 No Content + clears the refresh_token cookie
Action:
  - Read refresh token from HttpOnly cookie (401 if missing)
  - Hash the incoming refresh token
  - DELETE all sessions WHERE user_id = (SELECT user_id FROM refresh_tokens WHERE token_hash = ?)
  - All refresh tokens across all sessions cascade deleted via FK
  - Clear the refresh_token cookie

Notes:
  - If token not found or expired → 401
  - Logs out every device simultaneously
```

---

### Sessions

```md
GET /api/auth/sessions
Request:  (no body — refresh token read from cookie automatically)
Response: [
  {
    id: "uuid",
    device_info: { browser, platform, os, display_name },
    ip_address: "1.2.3.4",
    created_at: "2026-01-01T00:00:00Z",
    last_used_at: "2026-01-06T00:00:00Z"
  }
]
Action:
  - Read refresh token from cookie, look up user via token hash
  - Return all active sessions for that user

DELETE /api/auth/sessions/{id}
Auth: Bearer token (Authorization header)
Response: 204 No Content
Action:
  - Validate JWT, extract user ID
  - DELETE session WHERE id = ? AND user_id = ? (ownership enforced)
  - If not found or not owned → 404
```

---

## Token Rules

| Token | TTL | Server storage | Client storage | Notes |
| ----- | --- | -------------- | -------------- | ----- |
| Access | 15 min | None | JS memory (variable) | Short-lived JWT, not persisted |
| Refresh | 7 days | Postgres (SHA256 hash) | HttpOnly cookie | Rotated on every use, JS cannot read |
| OTP | 10 min | Redis (SHA256 hash) | None | Deleted after successful verification |

**Sessions:** one created per device login. Deleting a session cascades to its refresh token.
Each user can have multiple active sessions (one per device).

---

## Error Reference

| Scenario | Status | Message |
| -------- | ------ | ------- |
| User not found on login | 401 | `invalid credentials` |
| User already exists on registration | 409 | `user already exists` |
| OTP wrong or expired | 401 | `otp is invalid or expired, try again` |
| OTP too many attempts | 429 | `too many attempts, try again later` |
| Refresh token cookie missing | 401 | `missing refresh token` |
| Refresh token invalid or expired | 401 | `refresh token is invalid or expired` |
| Session not found or not owned | 404 | `session not found` |
| Missing/invalid Bearer token | 401 | `missing authorization header` / `invalid authorization header` |

---

## Frontend Integration

### Cookie behavior

The `refresh_token` cookie is `HttpOnly` — **JavaScript cannot read it**. The browser sends it automatically on requests to `/api/auth/*` when `credentials: 'include'` is set.

| Environment | SameSite | Secure | Notes |
| ----------- | -------- | ------ | ----- |
| Development (localhost) | `Lax` | `false` | HTTP ok |
| Production (cross-origin) | `None` | `true` | Requires HTTPS |

### Required fetch option

All auth endpoints that rely on the cookie **must** include `credentials: 'include'`:

```js
// Login
const { access_token, expires_at } = await fetch("/api/auth/token", {
  method: "POST",
  credentials: "include",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ identifier, otp, purpose })
}).then(r => r.json())

// Refresh (browser sends cookie automatically, no body needed)
const { access_token } = await fetch("/api/auth/token/refresh", {
  method: "POST",
  credentials: "include"
}).then(r => r.json())

// Logout
await fetch("/api/auth/logout", { method: "POST", credentials: "include" })
```

### Token storage

- **Access token** — store in a JS variable or React state. Do NOT put in `localStorage` (XSS risk).
- **Refresh token** — never touched by JS. The browser handles it entirely.

### Handling expiry

On any `401` from a protected endpoint, attempt a silent refresh then retry:

```js
async function fetchWithAuth(url, options = {}) {
  const res = await fetch(url, {
    ...options,
    headers: { ...options.headers, Authorization: `Bearer ${accessToken}` }
  })

  if (res.status === 401) {
    const refreshed = await fetch("/api/auth/token/refresh", {
      method: "POST",
      credentials: "include"
    })

    if (!refreshed.ok) { redirect to login }
    
    accessToken = (await refreshed.json()).access_token
    return fetch(url, { 
      ...options, 
      headers: {
        Authorization: `Bearer ${accessToken}`
      }
    })
  }

  return res
}
```

### CORS requirement

The API sets `Access-Control-Allow-Credentials: true` and `Access-Control-Allow-Origin` to the frontend's exact origin (not `*`). Configure the frontend origin via the `CORS_ALLOWED_ORIGINS` env var on the backend.

---

## Missing / To Add

### Nice to Have

- **Audit log** — record login attempts, logouts, and token refreshes with IP and device for security monitoring.

## Audit Logging

### Philosophy

- **Structured slog to stdout** — GCP Cloud Logging automatically ingests stdout JSON from Cloud Run/GKE. No SDK, no DB, no extra infra.
- **`audit=true` field** — every security event has this flag so GCP log queries can filter precisely.
- **Handler-level emission** — handlers know both the HTTP context (IP from `RequestInfo`) and the service outcome (success/error). Best place to combine both.
- **Fire-and-forget** — audit logging must never block or fail a request.

### GCP Cloud Logging queries (no config needed)

```txt
jsonPayload.audit=true
jsonPayload.audit=true AND jsonPayload.action="auth.failed"
jsonPayload.audit=true AND jsonPayload.ip="203.0.113.42"
jsonPayload.audit=true AND jsonPayload.user_id="3f01faf6-..."
```

Works automatically when the app runs on GCP with a JSON slog handler.

---

### Events to record

| Action | Emitted from | Key fields |
| ------ | ------------ | ---------- |
| `otp.requested` | `RequestOtp` handler | identifier, ip, purpose, status |
| `auth.success` | `Authenticate` handler | user_id, identifier, ip, purpose |
| `auth.failed` | `Authenticate` handler | identifier, ip, purpose, reason |
| `token.refreshed` | `RefreshToken` handler | user_id, ip |
| `token.refresh_failed` | `RefreshToken` handler | ip, reason |
| `logout` | `Logout` handler | user_id, ip |
| `logout.all` | `LogoutAll` handler | user_id, ip |
| `session.deleted` | `DeleteSession` handler | user_id, session_id, ip |

---

### New Files

**`internal/packages/eventlog/eventlog.go`**

```go
package eventlog

type EventLogger interface {
    Log(ctx context.Context, event Event)
}

type Event struct {
    Action   string         // e.g. "auth.success"
    Status   string         // "success" | "failure"
    UserID   string         // optional
    IP       string
    Reason   string         // optional, failure reason
    Fields   map[string]any // extra context
}

// Action constants
const (
    ActionOTPRequested    = "otp.requested"
    ActionAuthSuccess     = "auth.success"
    ActionAuthFailed      = "auth.failed"
    ActionTokenRefreshed  = "token.refreshed"
    ActionLogout          = "logout"
    ActionLogoutAll       = "logout.all"
    ActionSessionDeleted  = "session.deleted"
)
```

**`internal/packages/eventlog/slog.go`**

```go
type SlogEventLogger struct{ logger logging.Logger }

func (l *SlogEventLogger) Log(ctx context.Context, event Event) {
    args := []any{
        "audit",   true,
        "action",  event.Action,
        "status",  event.Status,
        "ip",      event.IP,
    }
    if event.UserID != "" { args = append(args, "user_id", event.UserID) }
    if event.Reason != "" { args = append(args, "reason", event.Reason) }
    for k, v := range event.Fields { args = append(args, k, v) }

    l.logger.Info("audit event", args...)
}
```

### Modified Files

**`internal/handlers/auth_handler.go`**

- Add `eventLogger eventlog.EventLogger` field to `AuthHandler`
- After each service call, emit the appropriate event using IP from `middlewares.GetRequestInfo(r.Context())`
- All `Log` calls are best-effort (no error handling needed — slog never returns errors)

**`internal/app/container.go`**

- Construct `eventlog.NewSlogEventLogger(logger)`
- Pass to `NewAuthHandler`

**`cmd/api/main.go`**

- No changes needed — EventLogger is constructed inside container wiring

---

## Wiring summary

```md
main.go
  └─ NewAppContainer(cfg, ..., redis)
        ├─ ratelimit.NewRedisRateLimiter(redis, cfg.RateLimit.StrictIP)   → auth routes
        ├─ eventlog.NewSlogEventLogger(logger)                            → AuthHandler
        └─ NewRouter()
              └─ r.With(RateLimitByIPMiddleware(...)).Post(...)
```

---

## What does NOT get rate limited (intentional)

- `GET /users/profile` — authenticated, low volume
- `DELETE /auth/sessions/{id}` — authenticated, low volume  
- Future `POST /picks` — once per match per user, natural limit via business logic
- Future `GET /tournaments`, `GET /standings` — read-only, cacheable later

---

## File checklist

| File | Action |
| ---- | ------ |
| `internal/packages/ratelimit/ratelimiter.go` | NEW |
| `internal/packages/ratelimit/redis.go` | NEW |
| `internal/packages/eventlog/eventlog.go` | NEW |
| `internal/packages/eventlog/slog.go` | NEW |
| `internal/infrastructure/middlewares/rate_limit_middleware.go` | NEW |
| `internal/infrastructure/config/config.go` | MODIFY — add `RateLimitConfig` |
| `internal/handlers/auth_handler.go` | MODIFY — add `eventLogger`, emit events |
| `internal/app/container.go` | MODIFY — wire limiters + eventlogger |
