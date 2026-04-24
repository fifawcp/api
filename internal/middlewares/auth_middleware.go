package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/infrastructure/auth"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/packages/httputils"
	"github.com/fifawcp/api/internal/services"
)

func Auth(
	authenticator auth.Authenticator,
	userService services.UserServiceInterface,
	logger logging.Logger,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				httputils.RespondWithError(w, http.StatusUnauthorized, ErrMissingAuthHeader)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				httputils.RespondWithError(w, http.StatusUnauthorized, ErrInvalidAuthHeader)
				return
			}

			token := parts[1]
			claims, err := authenticator.ValidateToken(token)
			if err != nil {
				httputils.RespondWithError(w, http.StatusUnauthorized, ErrInvalidToken)
				return
			}

			userID := claims.Subject

			user, err := userService.GetUser(r.Context(), userID)
			if err != nil {
				logger.Error(
					"failed to get user",
					"error", err,
					"userID", userID,
				)
				httputils.RespondWithError(w, http.StatusUnauthorized, ErrInvalidCredentials)
				return
			}

			ctx := context.WithValue(r.Context(), httpctx.AuthenticatedUserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
