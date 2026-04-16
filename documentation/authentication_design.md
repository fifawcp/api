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
