package middlewares

import (
	"net/http"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httputils"
)

func RequireAdminRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := httpctx.GetAuthenticatedUser(r.Context())

		if user.Role != domain.RoleAdmin {
			httputils.RespondWithError(w, http.StatusForbidden, domain.ErrForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
