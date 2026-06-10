package middlewares

import (
	"errors"
	"net/http"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/services"
)

func RequireTargetBoardMembership(boardMemberService services.BoardMemberServiceInterface) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			boardID := httpctx.GetBoardID(r.Context())
			targetUserID := httpctx.GetUserID(r.Context())

			if boardID == 0 || targetUserID == "" {
				httpx.InternalServerError(w, r, codeInternalServer, ErrInternalServer.Error())
				return
			}

			_, err := boardMemberService.GetBoardMember(r.Context(), boardID, targetUserID)
			if err != nil {
				switch {
				case errors.Is(err, domain.ErrBoardMemberNotFound):
					httpx.NotFound(w, r, codeBoardMemberNotFound, domain.ErrBoardMemberNotFound.Error())
				default:
					httpx.InternalServerError(w, r, codeInternalServer, ErrInternalServer.Error())
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
