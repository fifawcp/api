package middlewares

import (
	"context"
	"errors"
	"net/http"

	"github.com/fifawcp/api/internal/packages/httputils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func RequireValidUserID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userId")
		if _, err := uuid.Parse(userID); err != nil {
			httputils.RespondWithError(w, http.StatusBadRequest, errors.New("invalid user ID"))
			return
		}

		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
