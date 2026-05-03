package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/infrastructure/auth"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/services"
)

func OptionalAuth(
	authenticator auth.Authenticator,
	userService services.UserServiceInterface,
	logger logging.Logger,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := tryAuthenticate(r, authenticator, userService, logger)
			if user == nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), httpctx.AuthenticatedUserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func tryAuthenticate(
	r *http.Request,
	authenticator auth.Authenticator,
	userService services.UserServiceInterface,
	logger logging.Logger,
) *domain.User {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil
	}

	claims, err := authenticator.ValidateToken(parts[1])
	if err != nil {
		return nil
	}

	user, err := userService.GetUser(r.Context(), claims.Subject)
	if err != nil {
		logger.Error(
			"optional auth: failed to get user for valid token",
			logging.Error, err.Error(),
			logging.UserID, claims.Subject,
		)
		return nil
	}

	return user
}
