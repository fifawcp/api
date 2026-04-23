package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	httputils "github.com/fifawcp/api/internal/packages/httputils"
	"github.com/go-chi/chi/v5/middleware"
)

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

	var cooldownErr domain.OtpCooldownError
	var matchesNotFoundErr domain.MatchesNotFoundError

	// Map domain errors to HTTP responses
	switch {

	// Not found
	case
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrSessionNotFound),
		errors.Is(err, domain.ErrBoardNotFound),
		errors.Is(err, domain.ErrBoardMemberNotFound),
		errors.Is(err, domain.ErrMatchNotFound),
		errors.As(err, &matchesNotFoundErr):
		httputils.RespondWithError(w, http.StatusNotFound, err)

	// Conflict
	case
		errors.Is(err, domain.ErrUserAlreadyExists),
		errors.Is(err, domain.ErrUsernameAlreadyExists),
		errors.Is(err, domain.ErrBoardMemberAlreadyInBoard):
		httputils.RespondWithError(w, http.StatusConflict, err)

	// Unauthorized
	case
		errors.Is(err, domain.ErrOTPInvalidOrExpired),
		errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrRefreshTokenInvalidOrExpired),
		errors.Is(err, domain.ErrBoardInvalidJoinCode):
		httputils.RespondWithError(w, http.StatusUnauthorized, err)

	// Rate limit
	case
		errors.Is(err, domain.ErrOTPTooManyAttempts),
		errors.As(err, &cooldownErr):
		httputils.RespondWithError(w, http.StatusTooManyRequests, err)

	// Forbidden
	case
		errors.Is(err, domain.ErrForbidden),
		errors.Is(err, domain.ErrOAuthAccountNotVerified):
		httputils.RespondWithError(w, http.StatusForbidden, err)

	// Bad request
	case
		errors.Is(err, domain.ErrInvalidWinnerTeam),
		errors.Is(err, domain.ErrInvalidThirdPlaceTeam),
		errors.Is(err, domain.ErrThirdPlaceNotInConflict),
		errors.Is(err, domain.ErrThirdPlaceInvalidSelection),
		errors.Is(err, domain.ErrOAuthStateNotFound):
		httputils.RespondWithError(w, http.StatusBadRequest, err)

	// Bad gateway
	case
		errors.Is(err, domain.ErrMissingIDToken):
		httputils.RespondWithError(w, http.StatusBadGateway, err)

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
		httputils.RespondWithError(
			w,
			http.StatusInternalServerError,
			errors.New("internal server error"),
		)
	}
}
