package middlewares

import (
	"context"
	"errors"
	"net/http"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/packages/httputils"
	"github.com/fifawcp/api/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func RequireBoardMembership(boardMemberService services.BoardMemberServiceInterface) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			boardID := chi.URLParam(r, "boardId")
			user := GetAuthenticatedUser(r.Context())

			if _, err := uuid.Parse(boardID); err != nil {
				httputils.RespondWithError(w, http.StatusBadRequest, errors.New("invalid board ID"))
				return
			}

			boardMember, err := boardMemberService.GetBoardMember(r.Context(), boardID, user.ID)
			if err != nil {
				switch {
				case errors.Is(err, domain.ErrBoardNotFound):
					httputils.RespondWithError(w, http.StatusNotFound, err)
				case errors.Is(err, domain.ErrBoardMemberNotFound):
					httputils.RespondWithError(w, http.StatusForbidden, errors.New("not a member of this board"))
				default:
					httputils.RespondWithError(w, http.StatusInternalServerError, errors.New("internal server error"))
				}
				return
			}

			ctx := context.WithValue(r.Context(), BoardIDContextKey, boardID)
			ctx = context.WithValue(ctx, BoardMemberRoleContextKey, boardMember.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
