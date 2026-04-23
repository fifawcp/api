# OAuth Service Refactor Design

**Date:** 2026-04-23  
**Branch:** API-24  
**Scope:** `internal/services/oauth_service.go`, `internal/repositories/oauth_account_repository.go`, `internal/domain/oauth.go`

---

## Goal

Refactor the Google OAuth flow for clarity, debuggability, and correctness. The single flat `CompleteGoogleLogin` function (100+ lines, three login paths) becomes a readable orchestration of named private methods. All outstanding TODOs are addressed.

---

## Changes Overview

### 1. `internal/services/oauth_service.go`

#### `BeginGoogleLogin`

- Fix silently ignored `rand.Read` error — use `io.ReadFull(rand.Reader, ...)` and return the error.

#### `CompleteGoogleLogin` → orchestrator only

The public method becomes a flat, readable sequence of named calls:

```go
func (s *OAuthService) CompleteGoogleLogin(ctx, state, code, requestInfo) (*dtos.AuthenticationDto, string, error) {
    returnTo, err := s.validateAndConsumeOAuthState(ctx, state)
    idToken, err  := s.exchangeCodeForVerifiedIDToken(ctx, code)
    return s.resolveLoginForIDToken(ctx, idToken, returnTo, requestInfo)
}
```

#### Private methods added

| Method | Responsibility |
|---|---|
| `validateAndConsumeOAuthState(ctx, state)` | Retrieve and delete the OAuth state from Redis; returns `returnTo` |
| `exchangeCodeForVerifiedIDToken(ctx, code)` | Exchange auth code for OAuth2 token, extract and verify the raw ID token, return `*domain.IDToken` |
| `resolveLoginForIDToken(ctx, idToken, returnTo, requestInfo)` | Lookup oauth account; if found → `loginWithExistingOAuthAccount`; otherwise verify email is verified, then dispatch to link or register path |
| `loginWithExistingOAuthAccount(ctx, oauthAccount, returnTo, requestInfo)` | Fetch user by ID, issue authentication |
| `linkOAuthAccountToExistingUser(ctx, idToken, existingUser, returnTo, requestInfo)` | Create oauth account record linked to existing user, issue authentication |
| `registerNewUserViaOAuth(ctx, idToken, returnTo, requestInfo)` | Create user + oauth account atomically via repository transaction, issue authentication |

#### `generateUsernameFromEmail` (renamed from `slugify`)

- Rename to `generateUsernameFromEmail` for clarity.
- Add truncation guard: prefix `"google-"` (7 chars) + suffix `"-NNNN"` (up to 5 chars) = 12 chars overhead. Cap the email local-part at 38 chars to stay within the `CHAR(50)` column constraint.

#### Default name fallbacks

In `registerNewUserViaOAuth`, apply defaults before creating the user:
- `GivenName` empty → `"Google"`
- `FamilyName` empty → `"User"`

---

### 2. `internal/repositories/oauth_account_repository.go`

#### New method: `CreateUserWithOAuthAccount`

```go
func (r *OAuthAccountRepository) CreateUserWithOAuthAccount(
    ctx context.Context,
    user *domain.User,
    account *domain.OAuthAccount,
) error
```

Implementation:
1. `db.BeginTx(ctx, nil)`
2. INSERT into `users` — scan back `id`, `created_at`, `updated_at` into `user`
3. INSERT into `oauth_accounts` using `user.ID` — scan back `id`, `created_at`, `updated_at` into `account`
4. `tx.Commit()` — `defer tx.Rollback()` handles failure cases

Per-statement error attribution is preserved (each step returns a distinct error).

---

### 3. `internal/domain/oauth.go`

Add `CreateUserWithOAuthAccount` to the `OAuthAccountRepository` interface:

```go
type OAuthAccountRepository interface {
    CreateOAuthAccount(ctx context.Context, oauthAccount *OAuthAccount) error
    GetByProviderSub(ctx context.Context, provider string, providerSub string) (*OAuthAccount, error)
    CreateUserWithOAuthAccount(ctx context.Context, user *User, account *OAuthAccount) error
}
```

---

## TODOs Addressed

| TODO | Resolution |
|---|---|
| Ignored `rand.Read` error | Use `io.ReadFull(rand.Reader, ...)`, propagate error |
| `CompleteGoogleLogin` readability | Extracted into 5 named private methods |
| Default `GivenName`/`FamilyName` | Applied in `registerNewUserViaOAuth` before user creation |
| `slugify` CHAR(50) truncation risk | Capped email local-part at 38 chars in `generateUsernameFromEmail` |
| Orphaned user if auth issue fails | Eliminated — user+account created atomically in a transaction |
| Separate user+account inserts | Replaced by `CreateUserWithOAuthAccount` with `BEGIN`/`COMMIT` |
| `IssueAuthenticationForOAuth` TODO comments | Removed — method is clean after extraction |

## TODOs Deferred

| TODO | Reason |
|---|---|
| Username unique-violation retry / 409 | Separate concern; low collision probability with 4-digit suffix |
| `slugify` → shared utility package | Kept local to service for now; move when a second caller appears |

---

## Files Changed

| File | Change type |
|---|---|
| `internal/services/oauth_service.go` | Refactor |
| `internal/repositories/oauth_account_repository.go` | Add method |
| `internal/domain/oauth.go` | Extend interface |
