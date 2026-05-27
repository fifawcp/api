package httpx

import "errors"

const (
	codeValidationFailed   = "VALIDATION_FAILED"
	codeInvalidRequestBody = "INVALID_REQUEST_BODY"
	codeInvalidPageParam   = "INVALID_PAGE_PARAM"
	codeInvalidLimitParam  = "INVALID_LIMIT_PARAM"
	codeLimitOutOfRange    = "LIMIT_OUT_OF_RANGE"
)

var (
	errValidationFailed    = errors.New("validation failed")
	errRequestBodyEmpty    = errors.New("request body is empty")
	errRequestBodyTooLarge = errors.New("request body too large (max 1 MB)")
	errInvalidRequestBody  = errors.New("invalid request body")
	errMissingRefreshToken = errors.New("missing refresh token")
	errInvalidPageParam    = errors.New("invalid page parameter")
	errInvalidLimitParam   = errors.New("invalid limit parameter")
	errLimitOutOfRange     = errors.New("limit out of range")
)
