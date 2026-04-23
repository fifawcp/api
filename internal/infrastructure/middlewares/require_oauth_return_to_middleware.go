package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/packages/httputils"
)

func RequireOAuthReturnTo(logger logging.Logger, allowlist []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			returnTo := r.URL.Query().Get("return_to")
			if returnTo == "" {
				httputils.RespondWithError(w, http.StatusBadRequest, errors.New("return_to is a required query parameter"))
				return
			}

			// Validate return_to is a valid URL
			returnToURL, err := url.Parse(returnTo)
			if err != nil {
				httputils.RespondWithError(w, http.StatusBadRequest, errors.New("return_to is not a valid URL"))
				return
			}

			// Validate return_to is in the allowlist
			for _, allow := range allowlist {
				url := returnToURL.Scheme + "://" + returnToURL.Host
				if url == allow {
					ctx := context.WithValue(r.Context(), ReturnToContextKey, returnTo)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			httputils.RespondWithError(w, http.StatusBadRequest, errors.New("return_to URL is not in the allowlist"))
		})
	}
}
