package middlewares

import (
	"context"
	"net/http"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func RequireValidUserID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userId")
		if _, err := uuid.Parse(userID); err != nil {
			httpx.BadRequest(w, r, codeInvalidUserID, ErrInvalidUserID.Error())
			return
		}

		ctx := context.WithValue(r.Context(), httpctx.UserIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
