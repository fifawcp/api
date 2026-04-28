package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/auth"
	"github.com/fifawcp/api/internal/infrastructure/logging"
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
				httpx.Unauthorized(w, r, codeMissingAuthHeader, ErrMissingAuthHeader.Error())
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				httpx.Unauthorized(w, r, codeInvalidAuthHeader, ErrInvalidAuthHeader.Error())
				return
			}

			token := parts[1]
			claims, err := authenticator.ValidateToken(token)
			if err != nil {
				httpx.Unauthorized(w, r, codeInvalidToken, ErrInvalidToken.Error())
				return
			}

			userID := claims.Subject

			user, err := userService.GetUser(r.Context(), userID)
			if err != nil {
				logger.Error(
					"failed to get user",
					logging.Error, err.Error(),
					logging.UserID, userID,
				)
				httpx.Unauthorized(w, r, codeInvalidCredentials, ErrInvalidCredentials.Error())
				return
			}

			ctx := context.WithValue(r.Context(), httpctx.AuthenticatedUserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
