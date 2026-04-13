package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/ncondes/fifa-world-cup-pickems/internal/domain"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/logging"
)

var errEmptyBody = errors.New("request body is empty")
var errInvalidRequestBody = errors.New("invalid request body")
var errValidationFailed = errors.New("validation failed")
var errBodyTooLarge = errors.New("request body too large (max 1 MB)")
var errInternalServerError = errors.New("internal server error")

func errMalformedJSON(position int64) error {
	return fmt.Errorf("malformed JSON at position %d", position)
}

func errInvalidValueForField(field string, expectedType string) error {
	return fmt.Errorf("invalid value for field '%s' (expected %s)", field, expectedType)
}

func errUnknownField(fieldName string) error {
	return fmt.Errorf("unknown field '%s' in request body", fieldName)
}

func handleServiceError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	logger logging.Logger,
) {
	// Check if the request was cancelled
	if errors.Is(err, context.Canceled) {
		return
	}

	// Map domain errors to HTTP responses
	switch {

	// Not found
	case
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrSessionNotFound):
		respondWithError(w, http.StatusNotFound, err)

	// Conflict
	case
		errors.Is(err, domain.ErrUserAlreadyExists),
		errors.Is(err, domain.ErrUsernameAlreadyExists):
		respondWithError(w, http.StatusConflict, err)

	// Unauthorized
	case
		errors.Is(err, domain.ErrOTPInvalidOrExpired),
		errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrRefreshTokenInvalidOrExpired):
		respondWithError(w, http.StatusUnauthorized, err)

	// Rate limit
	case
		errors.Is(err, domain.ErrOTPTooManyAttempts),
		errors.As(err, &domain.OtpCooldownError{}):
		respondWithError(w, http.StatusTooManyRequests, err)

	// Internal server error
	default:
		// Log the error with request context
		logFields := []any{"error", err}
		if r != nil {
			logFields = append(logFields,
				"method", r.Method,
				"path", r.URL.Path,
				"request_id", r.Context().Value(middleware.RequestIDKey),
			)
		}
		logger.Error("internal server error", logFields...)

		// Always return generic error to client
		respondWithError(
			w,
			http.StatusInternalServerError,
			errInternalServerError,
		)
	}
}
