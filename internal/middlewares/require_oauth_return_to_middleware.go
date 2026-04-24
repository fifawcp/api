package middlewares

import (
	"context"
	"net/http"
	"net/url"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/httputils"
)

func RequireOAuthReturnTo(logger logging.Logger, allowlist []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			returnTo := r.URL.Query().Get("return_to")
			if returnTo == "" {
				httputils.RespondWithError(w, http.StatusBadRequest, ErrReturnToRequired)
				return
			}

			returnToURL, err := url.Parse(returnTo)
			if err != nil {
				httputils.RespondWithError(w, http.StatusBadRequest, ErrReturnToInvalidURL)
				return
			}

			for _, allow := range allowlist {
				url := returnToURL.Scheme + "://" + returnToURL.Host
				if url == allow {
					ctx := context.WithValue(r.Context(), httpctx.ReturnToContextKey, returnTo)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			httputils.RespondWithError(w, http.StatusBadRequest, ErrReturnToNotAllowed)
		})
	}
}
