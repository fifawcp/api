package middlewares

import "errors"

var (
	ErrMissingAuthHeader  = errors.New("missing authorization header")
	ErrInvalidAuthHeader  = errors.New("invalid authorization header")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrInvalidCredentials = errors.New("invalid credentials")

	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	ErrReturnToRequired   = errors.New("return_to is a required query parameter")
	ErrReturnToInvalidURL = errors.New("return_to is not a valid URL")
	ErrReturnToNotAllowed = errors.New("return_to URL is not in the allowlist")

	ErrInvalidUserID  = errors.New("invalid user ID")
	ErrInvalidBoardID = errors.New("invalid board ID")
	ErrInvalidMatchID = errors.New("invalid match ID")

	ErrNotBoardMember = errors.New("not a member of this board")
	ErrInternalServer = errors.New("internal server error")

	ErrOAuthFailed       = errors.New("oauth authorization failed")
	ErrMissingOAuthState = errors.New("missing oauth state")
	ErrMissingAuthCode   = errors.New("missing authorization code")
)
