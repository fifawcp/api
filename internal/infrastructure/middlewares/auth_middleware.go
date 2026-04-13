package middlewares

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ncondes/fifawcp/internal/infrastructure/auth"
	"github.com/ncondes/fifawcp/internal/infrastructure/logging"
	"github.com/ncondes/fifawcp/internal/packages/httputils"
	"github.com/ncondes/fifawcp/internal/services"
)

func AuthMiddleware(
	authenticator auth.Authenticator,
	userService services.UserServiceInterface,
	logger logging.Logger,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				httputils.RespondWithError(w, http.StatusUnauthorized, errors.New("missing authorization header"))
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				httputils.RespondWithError(w, http.StatusUnauthorized, errors.New("invalid authorization header"))
				return
			}

			token := parts[1]
			// Validate the token and get claims
			claims, err := authenticator.ValidateToken(token)
			if err != nil {
				httputils.RespondWithError(w, http.StatusUnauthorized, errors.New("invalid or expired token"))
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
				httputils.RespondWithError(w, http.StatusUnauthorized, errors.New("invalid credentials"))
				return
			}

			ctx := context.WithValue(r.Context(), authenticatedUserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
