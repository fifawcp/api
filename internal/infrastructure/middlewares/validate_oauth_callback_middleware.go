package middlewares

import (
	"context"
	"errors"
	"net/http"

	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/packages/httputils"
)

func ValidateOAuthCallback(logger logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.URL.Query().Get("error")
			if err != "" {
				logger.Error("oauth authorization failed", "error", err)
				httputils.RespondWithError(w, http.StatusBadRequest, errors.New("oauth authorization failed"))
				return
			}

			state := r.URL.Query().Get("state")
			if state == "" {
				httputils.RespondWithError(w, http.StatusBadRequest, errors.New("missing oauth state"))
				return
			}

			code := r.URL.Query().Get("code")
			if code == "" {
				httputils.RespondWithError(w, http.StatusBadRequest, errors.New("missing authorization code"))
				return
			}

			ctx := context.WithValue(r.Context(), OAuthStateContextKey, state)
			ctx = context.WithValue(ctx, OAuthCodeContextKey, code)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
