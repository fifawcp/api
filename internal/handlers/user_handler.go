package handlers

import (
	"net/http"

	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/services"
)

type UserHandler struct {
	userService services.UserServiceInterface
	logger      logging.Logger
	validator   *validator.Validator
}

func NewUserHandler(
	userService services.UserServiceInterface,
	logger logging.Logger,
	v *validator.Validator,
) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
		validator:   v,
	}
}

// GetProfile godoc
//
//	@Summary		Get authenticated user profile
//	@Description	Returns the full profile of the currently authenticated user resolved from the Bearer token.
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	httpx.Response{data=domain.User}	"Authenticated user profile"
//	@Failure		401	{object}	httpx.ErrorResponse					"Missing or invalid Bearer token"
//	@Security		BearerAuth
//	@Router			/users/profile [get]
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())
	httpx.RespondWithData(w, http.StatusOK, user)
}

// UpdateProfile godoc
//
//	@Summary		Update authenticated user profile
//	@Description	Partially updates the authenticated user's profile. Only provided fields are changed.
//	@Description	Email is changed through the dedicated email-change endpoints; role is never updatable.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dtos.UpdateUserDto					true	"Fields to update"
//	@Success		200		{object}	httpx.Response{data=domain.User}	"Updated user profile"
//	@Failure		400		{object}	httpx.ErrorResponse					"Invalid request body or validation error"
//	@Failure		401		{object}	httpx.ErrorResponse					"Missing or invalid Bearer token"
//	@Failure		409		{object}	httpx.ErrorResponse					"Username already taken"
//	@Failure		500		{object}	httpx.ErrorResponse					"Internal server error"
//	@Security		BearerAuth
//	@Router			/users/profile [patch]
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var body dtos.UpdateUserDto

	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	authenticatedUser := httpctx.GetAuthenticatedUser(r.Context())

	updatedUser, err := h.userService.UpdateUser(r.Context(), authenticatedUser.ID, &body)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, updatedUser)
}
