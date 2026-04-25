package httpx

import "errors"

const (
	codeValidationFailed   = "VALIDATION_FAILED"
	codeInvalidRequestBody = "INVALID_REQUEST_BODY"
)

var (
	errValidationFailed    = errors.New("validation failed")
	errRequestBodyEmpty    = errors.New("request body is empty")
	errRequestBodyTooLarge = errors.New("request body too large (max 1 MB)")
	errInvalidRequestBody  = errors.New("invalid request body")
	errMissingRefreshToken = errors.New("missing refresh token")
)
