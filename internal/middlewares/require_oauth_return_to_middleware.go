package middlewares

import (
	"context"
	"net/http"
	"net/url"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

func RequireOAuthReturnTo(logger logging.Logger, allowlist []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			returnTo := r.URL.Query().Get("return_to")
			if returnTo == "" {
				httpx.BadRequest(w, r, codeReturnToRequired, ErrReturnToRequired.Error())
				return
			}

			returnToURL, err := url.Parse(returnTo)
			if err != nil {
				httpx.BadRequest(w, r, codeReturnToInvalidURL, ErrReturnToInvalidURL.Error())
				return
			}

			for _, allow := range allowlist {
				origin := returnToURL.Scheme + "://" + returnToURL.Host
				if origin == allow {
					ctx := context.WithValue(r.Context(), httpctx.ReturnToContextKey, returnTo)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			httpx.BadRequest(w, r, codeReturnToNotAllowed, ErrReturnToNotAllowed.Error())
		})
	}
}
