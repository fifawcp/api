package middlewares

import (
	"context"
	"net/http"
	"strconv"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/go-chi/chi/v5"
)

func RequireValidMatchID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matchIDParam := chi.URLParam(r, "id")
		matchID, err := strconv.ParseInt(matchIDParam, 10, 64)
		if err != nil {
			httpx.BadRequest(w, r, codeInvalidMatchID, ErrInvalidMatchID.Error())
			return
		}

		ctx := context.WithValue(r.Context(), httpctx.MatchIDContextKey, matchID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
