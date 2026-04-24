package middlewares

import (
	"context"
	"net/http"
	"strconv"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httputils"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/go-chi/chi/v5"
)

func RequireValidMatchID(logger logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			matchIDParam := chi.URLParam(r, "id")
			matchID, err := strconv.ParseInt(matchIDParam, 10, 64)
			if err != nil {
				httputils.RespondWithError(w, http.StatusBadRequest, ErrInvalidMatchID)
				return
			}

			ctx := context.WithValue(r.Context(), httpctx.MatchIDContextKey, matchID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
