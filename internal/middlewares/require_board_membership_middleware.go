package middlewares

import (
	"context"
	"errors"
	"net/http"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httputils"
	"github.com/fifawcp/api/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func RequireBoardMembership(boardMemberService services.BoardMemberServiceInterface) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			boardID := chi.URLParam(r, "boardId")
			user := httpctx.GetAuthenticatedUser(r.Context())

			if _, err := uuid.Parse(boardID); err != nil {
				httputils.RespondWithError(w, http.StatusBadRequest, ErrInvalidBoardID)
				return
			}

			boardMember, err := boardMemberService.GetBoardMember(r.Context(), boardID, user.ID)
			if err != nil {
				switch {
				case errors.Is(err, domain.ErrBoardNotFound):
					httputils.RespondWithError(w, http.StatusNotFound, err)
				case errors.Is(err, domain.ErrBoardMemberNotFound):
					httputils.RespondWithError(w, http.StatusForbidden, ErrNotBoardMember)
				default:
					httputils.RespondWithError(w, http.StatusInternalServerError, ErrInternalServer)
				}
				return
			}

			ctx := context.WithValue(r.Context(), httpctx.BoardIDContextKey, boardID)
			ctx = context.WithValue(ctx, httpctx.BoardMemberRoleContextKey, boardMember.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
