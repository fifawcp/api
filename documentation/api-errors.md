# API Errors

Structured error responses with machine-readable codes, English messages, and a `request_id` for log correlation.

---

## Wire Format

### Standard error

```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "user not found",
    "request_id": "abc-123/xYz-000001"
  }
}
```

### Validation error

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "validation failed",
    "fields": {
      "email": {
        "code": "INVALID_EMAIL",
        "message": "must be a valid email address"
      },
      "first_name": {
        "code": "MAX_LENGTH",
        "message": "must be at most 255 characters",
        "params": { "max": 255 }
      },
      "tags": {
        "code": "MIN_ITEMS",
        "message": "must have at least 1 item",
        "params": { "min": 1 }
      }
    },
    "request_id": "abc-123/xYz-000001"
  }
}
```

`request_id` is always present and maps to the server log entry for the request.

---

## Go Types

```go
// internal/httpx/httpx.go
type ErrorResponse struct {
    Error APIError `json:"error"`
}

type APIError struct {
    Code      string                               `json:"code"`
    Message   string                               `json:"message"`
    RequestID string                               `json:"request_id,omitempty"`
    Fields    map[string]validator.ValidationField `json:"fields,omitempty"`
}

// internal/infrastructure/validator/validator.go
type ValidationField struct {
    Code    string         `json:"code"`
    Message string         `json:"message"`
    Params  map[string]any `json:"params,omitempty"`
}
```

---

## Error Code Inventory

### Domain / service errors

Handled in `internal/handlers/errors.go` via `handleServiceError`.

| Code | HTTP | Message |
| ---- | ---- | ------- |
| `USER_NOT_FOUND` | 404 | `user not found` |
| `SESSION_NOT_FOUND` | 404 | `session not found` |
| `BOARD_NOT_FOUND` | 404 | `board not found` |
| `BOARD_MEMBER_NOT_FOUND` | 404 | `board member not found` |
| `MATCH_NOT_FOUND` | 404 | `match not found` |
| `MATCHES_NOT_FOUND` | 404 | `one or more matches not found` |
| `OAUTH_ACCOUNT_NOT_FOUND` | 404 | `oauth account not found` |
| `USER_ALREADY_EXISTS` | 409 | `user already exists` |
| `USERNAME_ALREADY_EXISTS` | 409 | `username is already taken` |
| `BOARD_ALREADY_EXISTS` | 409 | `board already exists` |
| `BOARD_USER_ALREADY_IN_BOARD` | 409 | `user is already in this board` |
| `BOARD_MEMBER_ALREADY_IN_BOARD` | 409 | `user is already a member of this board` |
| `MAX_BOARD_MEMBERS_EXCEEDED` | 409 | `maximum board members exceeded` |
| `OTP_INVALID_OR_EXPIRED` | 401 | `OTP is invalid or expired` |
| `INVALID_CREDENTIALS` | 401 | `invalid credentials` |
| `REFRESH_TOKEN_NOT_FOUND` | 401 | `refresh token not found` |
| `REFRESH_TOKEN_INVALID_OR_EXPIRED` | 401 | `refresh token is invalid or expired` |
| `BOARD_INVALID_JOIN_CODE` | 401 | `invalid or expired board join code` |
| `OTP_TOO_MANY_ATTEMPTS` | 429 | `too many OTP attempts` |
| `OTP_COOLDOWN` | 429 | `please wait {n} seconds before requesting a new code` |
| `FORBIDDEN` | 403 | `insufficient permissions` |
| `OAUTH_ACCOUNT_NOT_VERIFIED` | 403 | `oauth account not verified` |
| `INVALID_WINNER_TEAM` | 400 | `winner team must be either home or away team` |
| `INVALID_THIRD_PLACE_TEAM` | 400 | `team is not a valid third-place team` |
| `THIRD_PLACE_NOT_IN_CONFLICT` | 400 | `third-place match is not in conflict` |
| `THIRD_PLACE_INVALID_SELECTION` | 400 | `invalid third-place team selection` |
| `OAUTH_STATE_NOT_FOUND` | 400 | `oauth state not found` |
| `INVALID_GROUP_CODE` | 400 | `invalid group code` |
| `INVALID_STAGE_CODE` | 400 | `invalid stage code` |
| `INVALID_STATUS` | 400 | `invalid status` |
| `INVALID_FIFA_CODE` | 400 | `invalid fifa code` |
| `INVALID_DATE_RANGE` | 400 | `from_date must be before or equal to to_date` |
| `MISSING_ID_TOKEN` | 502 | `missing identity token` |
| `INTERNAL_SERVER_ERROR` | 500 | `internal server error` |

### Middleware errors

Written directly in each middleware file. Defined as constants and vars in `internal/middlewares/errors.go`.

| Code | HTTP | Message |
| ---- | ---- | ------- |
| `MISSING_AUTH_HEADER` | 401 | `missing authorization header` |
| `INVALID_AUTH_HEADER` | 401 | `invalid authorization header` |
| `INVALID_TOKEN` | 401 | `invalid or expired token` |
| `INVALID_CREDENTIALS` | 401 | `invalid credentials` |
| `RATE_LIMIT_EXCEEDED` | 429 | `rate limit exceeded` |
| `NOT_BOARD_MEMBER` | 403 | `not a member of this board` |
| `FORBIDDEN` | 403 | `insufficient permissions` |
| `BOARD_NOT_FOUND` | 404 | `board not found` |
| `INVALID_USER_ID` | 400 | `invalid user ID` |
| `INVALID_BOARD_ID` | 400 | `invalid board ID` |
| `INVALID_MATCH_ID` | 400 | `invalid match ID` |
| `RETURN_TO_REQUIRED` | 400 | `return_to is a required query parameter` |
| `RETURN_TO_INVALID_URL` | 400 | `return_to is not a valid URL` |
| `RETURN_TO_NOT_ALLOWED` | 400 | `return_to URL is not in the allowlist` |
| `OAUTH_FAILED` | 400 | `oauth authorization failed` |
| `MISSING_OAUTH_STATE` | 400 | `missing oauth state` |
| `MISSING_AUTH_CODE` | 400 | `missing authorization code` |
| `INTERNAL_SERVER_ERROR` | 500 | `internal server error` |

### Request body errors

All use code `INVALID_REQUEST_BODY` with a descriptive message. Produced by `httpx.ReadAndValidateJSON` before the handler runs.

| Scenario | Message |
| -------- | ------- |
| Empty body | `request body is empty` |
| Malformed JSON | `malformed JSON at position {n}` |
| Wrong field type | `invalid value for field '{name}' (expected {type})` |
| Unknown field | `unknown field '{name}' in request body` |
| Body over 1 MB | `request body too large (max 1 MB)` |
| Other decode error | `invalid request body` |

---

## Frontend Integration

### Reading an error

```ts
const res = await fetch("/api/auth/token", { ... })

if (!res.ok) {
  const { error } = await res.json()
  // error.code    → machine-readable, use for i18n lookup
  // error.message → English fallback
  // error.request_id → include in bug reports
}
```

### Reading a validation error

```ts
if (res.status === 400) {
  const { error } = await res.json()

  if (error.code === "VALIDATION_FAILED") {
    for (const [field, detail] of Object.entries(error.fields)) {
      // detail.code    → e.g. "MAX_LENGTH"
      // detail.message → e.g. "must be at most 255 characters"
      // detail.params  → e.g. { max: 255 } (present when relevant)
    }
  }
}
```

### i18n strategy

`error.code` and `error.fields[field].code` are the stable keys for translation lookups. The `message` values are English and serve as developer debugging aids and i18n fallbacks — they are not meant to be shown verbatim to end users.

```ts
const t = {
  USER_NOT_FOUND: "Usuario no encontrado",
  VALIDATION_FAILED: "Por favor corrige los errores del formulario",
  // ...
}

const label = t[error.code] ?? error.message
```