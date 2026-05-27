package middlewares

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/services"
	"github.com/go-chi/chi/v5"
)

func RequireBoardMembership(boardMemberService services.BoardMemberServiceInterface) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			boardIDStr := chi.URLParam(r, "boardId")
			user := httpctx.GetAuthenticatedUser(r.Context())

			boardID, err := strconv.ParseInt(boardIDStr, 10, 64)
			if err != nil {
				httpx.BadRequest(w, r, codeInvalidBoardID, ErrInvalidBoardID.Error())
				return
			}

			boardMember, err := boardMemberService.GetBoardMember(r.Context(), boardID, user.ID)
			if err != nil {
				switch {
				case errors.Is(err, domain.ErrBoardNotFound):
					httpx.NotFound(w, r, codeBoardNotFound, domain.ErrBoardNotFound.Error())
				case errors.Is(err, domain.ErrBoardMemberNotFound):
					httpx.Forbidden(w, r, codeNotBoardMember, ErrNotBoardMember.Error())
				default:
					httpx.InternalServerError(w, r, codeInternalServer, ErrInternalServer.Error())
				}
				return
			}

			ctx := context.WithValue(r.Context(), httpctx.BoardIDContextKey, boardID)
			ctx = context.WithValue(ctx, httpctx.BoardMemberRoleContextKey, boardMember.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
