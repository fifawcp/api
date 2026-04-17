package httputils

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
)

const refreshTokenCookieName = "refresh_token"
const maxBodySizeInBytes = 1_048_576 // 1 MB

type Response struct {
	Data any `json:"data,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error" example:"error message"`
	Details any    `json:"details,omitempty" swaggertype:"object"` // For validation errors
}

// Response utilities
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func RespondWithData(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, Response{
		Data: data,
	})
}

func RespondWithError(w http.ResponseWriter, status int, err error) {
	WriteJSON(w, status, ErrorResponse{
		Error: err.Error(),
	})
}

func RespondWithValidationError(w http.ResponseWriter, details any) {
	WriteJSON(w, http.StatusBadRequest, ErrorResponse{
		Error:   "validation failed",
		Details: details,
	})
}

// Request utilities
func ReadJSON(w http.ResponseWriter, r *http.Request, data any) error {
	// Limit the size of the request body to 1MB to prevent abuse
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySizeInBytes)

	decoder := json.NewDecoder(r.Body)
	// Prevent injection of additional fields
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(data); err != nil {
		return parseJSONError(err)
	}

	return nil
}

func ReadAndValidateJSON(
	w http.ResponseWriter,
	r *http.Request,
	data any,
	v *validator.Validator,
) error {
	if err := ReadJSON(w, r, data); err != nil {
		RespondWithError(w, http.StatusBadRequest, err)
		return err
	}

	if validationErrors := v.ValidateStruct(data); len(validationErrors) > 0 {
		RespondWithValidationError(w, validationErrors)
		return errors.New("validation failed")
	}

	return nil
}

func parseJSONError(err error) error {
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var maxBytesError *http.MaxBytesError

	switch {
	// Empty body
	case errors.Is(err, io.EOF):
		return errors.New("request body is empty")

	// Syntax error in JSON
	case errors.As(err, &syntaxError):
		return errors.New("malformed JSON at position " + strconv.FormatInt(syntaxError.Offset, 10))

	// Wrong type for field
	case errors.As(err, &unmarshalTypeError):
		return errors.New("invalid value for field '" + unmarshalTypeError.Field + "' (expected " + unmarshalTypeError.Type.String() + ")")

	// Unknown field
	case strings.HasPrefix(err.Error(), "json: unknown field"):
		// Extract field name from error message
		fieldName := extractFieldName(err.Error())
		return errors.New("unknown field '" + fieldName + "' in request body")

	// Body too large
	case errors.As(err, &maxBytesError):
		return errors.New("request body too large (max 1 MB)")

	default:
		return errors.New("invalid request body")
	}
}

func extractFieldName(errMsg string) string {
	// "json: unknown field \"email\"" -> "email"
	parts := strings.Split(errMsg, "\"")
	if len(parts) >= 2 {
		return parts[1]
	}

	return "unknown"
}

// Cookie utilities
func SetRefreshTokenCookie(
	w http.ResponseWriter,
	token string,
	expiry time.Time,
	secure bool,
) {
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
		return "", errors.New("missing refresh token")
	}

	return cookie.Value, nil
}

func ParseStringSliceParam(r *http.Request, key string) []string {
	values := []string{}

	for _, value := range r.URL.Query()[key] {
		// Each value might be comma-separated
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
