package handlers

import (
	"net/http"

	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/packages/httputils"
	"github.com/fifawcp/api/internal/services"
)

type UserHandler struct {
	userService services.UserServiceInterface
	logger      logging.Logger
}

func NewUserHandler(
	userService services.UserServiceInterface,
	logger logging.Logger,
) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// GetProfile godoc
//
//	@Summary		Get authenticated user profile
//	@Description	Returns the full profile of the currently authenticated user resolved from the Bearer token.
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	httputils.Response{data=domain.User}	"Authenticated user profile"
//	@Failure		401	{object}	httputils.ErrorResponse					"Missing or invalid Bearer token"
//	@Security		BearerAuth
//	@Router			/users/profile [get]
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())
	httputils.RespondWithData(w, http.StatusOK, user)
}
