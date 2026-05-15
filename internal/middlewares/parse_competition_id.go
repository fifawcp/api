package middlewares

import (
	"context"
	"net/http"
	"strconv"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/go-chi/chi/v5"
)

func ParseCompetitionID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		competitionIDStr := chi.URLParam(r, "competitionId")
		competitionID, err := strconv.ParseInt(competitionIDStr, 10, 64)
		if err != nil {
			httpx.BadRequest(w, r, codeInvalidBoardID, "invalid competition id")
			return
		}

		ctx := context.WithValue(r.Context(), httpctx.CompetitionIDContextKey, competitionID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
