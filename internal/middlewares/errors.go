package middlewares

import "errors"

const (
	// 401 Unauthorized
	codeMissingAuthHeader  = "MISSING_AUTH_HEADER"
	codeInvalidAuthHeader  = "INVALID_AUTH_HEADER"
	codeInvalidToken       = "INVALID_TOKEN"
	codeInvalidCredentials = "INVALID_CREDENTIALS"

	// 429 Too Many Requests
	codeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"

	// 403 Forbidden
	codeNotBoardMember = "NOT_BOARD_MEMBER"
	codeForbidden      = "FORBIDDEN"

	// 404 Not Found
	codeBoardNotFound        = "BOARD_NOT_FOUND"
	codeBoardMemberNotFound  = "BOARD_MEMBER_NOT_FOUND"

	// 400 Bad Request
	codeReturnToRequired   = "RETURN_TO_REQUIRED"
	codeReturnToInvalidURL = "RETURN_TO_INVALID_URL"
	codeReturnToNotAllowed = "RETURN_TO_NOT_ALLOWED"
	codeInvalidUserID      = "INVALID_USER_ID"
	codeInvalidBoardID     = "INVALID_BOARD_ID"
	codeInvalidMatchID     = "INVALID_MATCH_ID"
	codeOAuthFailed        = "OAUTH_FAILED"
	codeMissingOAuthState  = "MISSING_OAUTH_STATE"
	codeMissingAuthCode    = "MISSING_AUTH_CODE"

	// 500 Internal Server Error
	codeInternalServer = "INTERNAL_SERVER_ERROR"
)

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

	ErrOAuthFailed       = errors.New("oauth authorization failed")
	ErrMissingOAuthState = errors.New("missing oauth state")
	ErrMissingAuthCode   = errors.New("missing authorization code")

	ErrInternalServer = errors.New("internal server error")
)
