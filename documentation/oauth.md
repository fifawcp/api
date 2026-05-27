# OAuth sign-in (Google)

OAuth 2.0 Authorization Code Flow + OIDC. Google is the only provider. Session/refresh token contract is identical to the OTP flow — see [authentication.md](./authentication.md). GCP setup: [GOOGLE.md](../GOOGLE.md).

---

## Routes

| Method | Path | Middleware |
| ------ | ---- | ---------- |
| `GET` | `/api/oauth/google` | `RateLimitByIP (Strict)` → `RequireOAuthReturnTo` |
| `GET` | `/api/oauth/google/callback` | `RateLimitByIP (Moderate)` → `ValidateOAuthCallback` → `RequestInfo` |

---

## Config

| Variable | Purpose |
| -------- | ------- |
| `GOOGLE_OAUTH_CLIENT_ID` / `GOOGLE_OAUTH_CLIENT_SECRET` | GCP OAuth web client credentials |
| `GOOGLE_OAUTH_REDIRECT_URL` | Must match the authorized redirect URI in GCP |
| `GOOGLE_OAUTH_RETURN_TO_ALLOWLIST` | Comma-separated `scheme://host` values allowed as `return_to` |
| `GOOGLE_OAUTH_STATE_TTL` | Redis TTL for the CSRF state record |
| `GOOGLE_OAUTH_ISSUER` | OIDC issuer — default `https://accounts.google.com` |

---

## Start (`GET /api/oauth/google`)

`RequireOAuthReturnTo` validates `return_to` is present and its `scheme://host` is in the allowlist.

`OAuthService.BeginGoogleLogin`:

1. Generates 32 random bytes → base64 → `state`
2. Stores `state → return_to` in Redis with TTL (`oauth:state:{state}`)
3. Builds and returns the Google authorization URL

→ **HTTP 307** to Google.

---

## Callback (`GET /api/oauth/google/callback`)

`ValidateOAuthCallback` rejects requests with a `?error=` param (user denied) or missing `state`/`code`.

`OAuthService.CompleteGoogleLogin`:

1. **Validate + consume state** — `GETDEL` from Redis. Unknown or replayed state → `ErrOAuthStateNotFound`. Also retrieves the original `return_to`.

2. **Exchange code → verified ID token**
   - `OAuth2Client.ExchangeCodeForToken` calls Google's token endpoint, extracts the raw `id_token` JWT
   - `IDTokenVerifier.Verify` checks signature, `iss`/`aud`/`exp`, and parses claims into `domain.IDToken`

3. **Resolve login** — see section below

→ Sets refresh token cookie → **HTTP 302** to `return_to` (from Redis, not from the query string).

The client then calls `POST /api/auth/token/refresh` with `credentials: 'include'` to get an access token.

---

## Login resolution (`resolveLoginForIDToken`)

```md
GetByProviderSub(provider, sub)
    │
    ├── found ─────────────────► loginWithExistingOAuthAccount
    │                                GetUserByID → IssueAuthentication
    └── not found
              │
              ├── email not verified ──► ErrOAuthAccountNotVerified (403)
              └── email verified
                        │
                        GetUserByIdentifier(email)
                        │
                        ├── found ─────► linkOAuthAccountToExistingUser
                        │                    CreateOAuthAccount → IssueAuthentication
                        └── not found ─► registerNewUserViaOAuth
                                             CreateUserWithOAuthAccount (tx) → IssueAuthentication
```

**Path 1** — returning OAuth user. The `oauth_accounts` row exists, load and authenticate.

**Path 2** — user exists (created via OTP) but no OAuth link yet. Creates the link, then authenticates. Next login takes Path 1.

**Path 3** — brand new user. Creates both `users` and `oauth_accounts` rows in a single transaction so a partial failure can't leave an orphaned user.

---

## Design decisions

**CSRF via stateful state.** The `state` is stored server-side in Redis and consumed atomically on callback (`GETDEL`). An unknown, expired, or replayed state is rejected immediately. `return_to` is stored alongside the state — it cannot be tampered with between the start and callback steps.

**Atomic user + account creation.** Path 3 runs inside a `BEGIN`/`COMMIT` transaction. If either insert fails, both roll back.

---

## Errors

| Error | Cause | Status |
| ----- | ----- | ------ |
| `ErrOAuthStateNotFound` | Unknown/expired/replayed state | 400 |
| `ErrMissingIDToken` | Google response missing `id_token` | 502 |
| `ErrOAuthAccountNotVerified` | Google email not verified | 403 |
| Middleware `?error=` param | User denied Google authorization | 400 |
| Middleware missing `state`/`code` | Malformed callback URL | 400 |
| `return_to` not in allowlist | Invalid start URL | 400 |
