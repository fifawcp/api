package middlewares

import (
	"context"
	"net/http"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httputils"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

func ValidateOAuthCallback(logger logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.URL.Query().Get("error")
			if err != "" {
				logger.Error("oauth authorization failed", "error", err)
				httputils.RespondWithError(w, http.StatusBadRequest, ErrOAuthFailed)
				return
			}

			state := r.URL.Query().Get("state")
			if state == "" {
				httputils.RespondWithError(w, http.StatusBadRequest, ErrMissingOAuthState)
				return
			}

			code := r.URL.Query().Get("code")
			if code == "" {
				httputils.RespondWithError(w, http.StatusBadRequest, ErrMissingAuthCode)
				return
			}

			ctx := context.WithValue(r.Context(), httpctx.OAuthStateContextKey, state)
			ctx = context.WithValue(ctx, httpctx.OAuthCodeContextKey, code)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
