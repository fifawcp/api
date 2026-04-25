package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/go-chi/chi/v5/middleware"
)

const refreshTokenCookieName = "refresh_token"
const maxBodySizeInBytes = 1_048_576 // 1 MB

// TODO: see if we can like customize the field of each response
type Response struct {
	Data any `json:"data,omitempty"`
}

type APIError struct {
	Code      string                               `json:"code"`
	Message   string                               `json:"message"`
	RequestID string                               `json:"request_id,omitempty"`
	Fields    map[string]validator.ValidationField `json:"fields,omitempty"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

// ---------------------------------------------------------------------------
// Response utilities
// ---------------------------------------------------------------------------
func RespondWithData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, Response{Data: data})
}

func RespondWithError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	apiErr := APIError{
		Code:    code,
		Message: message,
	}

	if r != nil {
		apiErr.RequestID = middleware.GetReqID(r.Context())
	}

	writeJSON(w, status, ErrorResponse{Error: apiErr})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// ---------------------------------------------------------------------------
// Status helpers
// ---------------------------------------------------------------------------
func NotFound(w http.ResponseWriter, r *http.Request, code, msg string) {
	RespondWithError(w, r, http.StatusNotFound, code, msg)
}

func Conflict(w http.ResponseWriter, r *http.Request, code, msg string) {
	RespondWithError(w, r, http.StatusConflict, code, msg)
}

func Unauthorized(w http.ResponseWriter, r *http.Request, code, msg string) {
	RespondWithError(w, r, http.StatusUnauthorized, code, msg)
}

func Forbidden(w http.ResponseWriter, r *http.Request, code, msg string) {
	RespondWithError(w, r, http.StatusForbidden, code, msg)
}

func BadRequest(w http.ResponseWriter, r *http.Request, code, msg string) {
	RespondWithError(w, r, http.StatusBadRequest, code, msg)
}

func TooManyRequests(w http.ResponseWriter, r *http.Request, code, msg string) {
	RespondWithError(w, r, http.StatusTooManyRequests, code, msg)
}

func InternalServerError(w http.ResponseWriter, r *http.Request, code, msg string) {
	RespondWithError(w, r, http.StatusInternalServerError, code, msg)
}

func BadGateway(w http.ResponseWriter, r *http.Request, code, msg string) {
	RespondWithError(w, r, http.StatusBadGateway, code, msg)
}

// ---------------------------------------------------------------------------
// Request utilities
// ---------------------------------------------------------------------------
func ReadAndValidateJSON(
	w http.ResponseWriter,
	r *http.Request,
	data any,
	v *validator.Validator,
) error {
	if err := readJSON(w, r, data); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, codeInvalidRequestBody, err.Error())
		return err
	}

	if validationErrors := v.ValidateStruct(data); len(validationErrors) > 0 {
		respondWithValidationError(w, r, validationErrors)
		return errValidationFailed
	}

	return nil
}

func readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySizeInBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(data); err != nil {
		return parseJSONError(err)
	}

	return nil
}

func respondWithValidationError(w http.ResponseWriter, r *http.Request, fields map[string]validator.ValidationField) {
	apiErr := APIError{
		Code:    codeValidationFailed,
		Message: errValidationFailed.Error(),
		Fields:  fields,
	}

	if r != nil {
		apiErr.RequestID = middleware.GetReqID(r.Context())
	}

	writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: apiErr})
}

func parseJSONError(err error) error {
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var maxBytesError *http.MaxBytesError

	switch {
	case errors.Is(err, io.EOF):
		return errRequestBodyEmpty
	case errors.As(err, &syntaxError):
		return fmt.Errorf("malformed JSON at position %d", syntaxError.Offset)
	case errors.As(err, &unmarshalTypeError):
		return fmt.Errorf("invalid value for field '%s' (expected %s)", unmarshalTypeError.Field, unmarshalTypeError.Type.String())
	case strings.HasPrefix(err.Error(), "json: unknown field"):
		return fmt.Errorf("unknown field '%s' in request body", extractFieldName(err.Error()))
	case errors.As(err, &maxBytesError):
		return errRequestBodyTooLarge
	default:
		return errInvalidRequestBody
	}
}

func extractFieldName(errMsg string) string {
	parts := strings.Split(errMsg, "\"")
	if len(parts) >= 2 {
		return parts[1]
	}

	return "unknown"
}

// ---------------------------------------------------------------------------
// Cookie utilities
// ---------------------------------------------------------------------------
func SetRefreshTokenCookie(w http.ResponseWriter, token string, expiry time.Time, secure bool) {
	sameSite := http.SameSiteLaxMode

	if secure {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/api/auth",
		Expires:  expiry,
		MaxAge:   int(time.Until(expiry).Seconds()),
	})
}

func ClearRefreshTokenCookie(w http.ResponseWriter, secure bool) {
	sameSite := http.SameSiteLaxMode

	if secure {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		HttpOnly: true,
		SameSite: sameSite,
		Path:     "/api/auth",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func GetRefreshTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil {
		return "", errMissingRefreshToken
	}

	return cookie.Value, nil
}

// ---------------------------------------------------------------------------
// Query param utilities
// ---------------------------------------------------------------------------
func ParseStringSliceParam(r *http.Request, key string) []string {
	values := []string{}
	for _, value := range r.URL.Query()[key] {
		for item := range strings.SplitSeq(value, ",") {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				values = append(values, trimmed)
			}
		}
	}

	return values
}

func ParseDateParam(r *http.Request, key string) (*time.Time, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return nil, fmt.Errorf("invalid '%s' date format, expected RFC3339", key)
	}

	return &parsed, nil
}

func ParseInt64Param(r *http.Request, key string) (*int64, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid '%s' parameter, expected int64", key)
	}

	return &parsed, nil
}
