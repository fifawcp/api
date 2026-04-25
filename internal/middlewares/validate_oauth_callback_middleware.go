package middlewares

import (
	"context"
	"net/http"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

func ValidateOAuthCallback(logger logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.URL.Query().Get("error")
			if err != "" {
				logger.Error("oauth authorization failed", "error", err)
				httpx.BadRequest(w, r, codeOAuthFailed, ErrOAuthFailed.Error())
				return
			}

			state := r.URL.Query().Get("state")
			if state == "" {
				httpx.BadRequest(w, r, codeMissingOAuthState, ErrMissingOAuthState.Error())
				return
			}

			code := r.URL.Query().Get("code")
			if code == "" {
				httpx.BadRequest(w, r, codeMissingAuthCode, ErrMissingAuthCode.Error())
				return
			}

			ctx := context.WithValue(r.Context(), httpctx.OAuthStateContextKey, state)
			ctx = context.WithValue(ctx, httpctx.OAuthCodeContextKey, code)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
