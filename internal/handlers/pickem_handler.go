package handlers

import (
	"net/http"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/services"
)

type PickemHandler struct {
	pickemService services.PickemServiceInterface
	logger        logging.Logger
	validator     *validator.Validator
}

func NewPickemHandler(
	pickemService services.PickemServiceInterface,
	logger logging.Logger,
	validator *validator.Validator,
) *PickemHandler {
	return &PickemHandler{
		pickemService: pickemService,
		logger:        logger,
		validator:     validator,
	}
}

// GetUserPickem returns the user's full pickem state.
//
//	@Summary		Get user's pickem
//	@Description	Returns group picks, best thirds, projected bracket, progress, and lock status.
//	@Tags			pickems
//	@Produce		json
//	@Success		200	{object}	domain.UserPickem	"User's pickem state"
//	@Failure		401	{object}	httpx.ErrorResponse	"Missing or invalid Bearer token"
//	@Security		BearerAuth
//	@Router			/pickems [get]
func (h *PickemHandler) GetUserPickem(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	pickem, err := h.pickemService.GetUserPickem(r.Context(), user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, pickem)
}

// SaveGroupPicks saves (overwrites) the user's group order picks and syncs each group's
// lock state from the per-group `locked` flag. Changing a group's order cascade-clears
// best-thirds + bracket; toggling only a lock does not.
//
//	@Summary	Save group picks
//	@Tags		pickems
//	@Accept		json
//	@Produce	json
//	@Param		body	body		dtos.SaveGroupPicksDto	true	"Group picks"
//	@Success	200		{object}	domain.UserPickem		"User's pickem state"
//	@Failure	400		{object}	httpx.ErrorResponse		"Invalid request body"
//	@Failure	401		{object}	httpx.ErrorResponse		"Missing or invalid Bearer token"
//	@Security	BearerAuth
//	@Router		/pickems/groups [put]
func (h *PickemHandler) SaveGroupPicks(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	var body dtos.SaveGroupPicksDto
	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	picks := make([]*domain.UserGroupPick, 0, len(body.GroupPicks)*4)
	lockedCodes := make([]string, 0, len(body.GroupPicks))
	for _, g := range body.GroupPicks {
		for i, code := range g.TeamFifaCodes {
			picks = append(picks, &domain.UserGroupPick{
				TeamFifaCode:      code,
				TeamGroupCode:     g.GroupCode,
				PredictedPosition: i + 1,
			})
		}
		if g.Locked {
			lockedCodes = append(lockedCodes, g.GroupCode)
		}
	}

	if err := h.pickemService.SaveGroupPicks(r.Context(), user.ID, picks, lockedCodes); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	pickem, err := h.pickemService.GetUserPickem(r.Context(), user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, pickem)
}

// SaveBestThirds saves the user's 8 best-third picks. Requires all 12 group picks
// to be saved and complete before this endpoint accepts input.
//
//	@Summary	Save best-third picks
//	@Tags		pickems
//	@Accept		json
//	@Produce	json
//	@Param		body	body		dtos.SaveBestThirdsDto	true	"8 best-third team codes"
//	@Success	200		{object}	domain.UserPickem		"User's pickem state"
//	@Failure	400		{object}	httpx.ErrorResponse		"Invalid request body"
//	@Failure	401		{object}	httpx.ErrorResponse		"Missing or invalid Bearer token"
//	@Security	BearerAuth
//	@Router		/pickems/best-thirds [put]
func (h *PickemHandler) SaveBestThirds(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	var body dtos.SaveBestThirdsDto
	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	if err := h.pickemService.SaveBestThirds(r.Context(), user.ID, body.TeamFifaCodes); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	pickem, err := h.pickemService.GetUserPickem(r.Context(), user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, pickem)
}

// SaveBracketPicks saves the user's bracket picks (winner per knockout match).
// Requires complete group + best-third picks. Strict 32-pick submission.
//
//	@Summary	Save bracket picks
//	@Tags		pickems
//	@Accept		json
//	@Produce	json
//	@Param		body	body		dtos.SaveBracketPicksDto	true	"32 bracket picks"
//	@Success	200		{object}	domain.UserPickem			"User's pickem state"
//	@Failure	400		{object}	httpx.ErrorResponse			"Invalid request body"
//	@Failure	401		{object}	httpx.ErrorResponse			"Missing or invalid Bearer token"
//	@Security	BearerAuth
//	@Router		/pickems/bracket [put]
func (h *PickemHandler) SaveBracketPicks(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	var body dtos.SaveBracketPicksDto
	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	picks := make([]*domain.UserBracketPick, 0, len(body.BracketPicks))
	for _, p := range body.BracketPicks {
		picks = append(picks, &domain.UserBracketPick{
			UserID:       user.ID,
			MatchID:      p.MatchID,
			TeamFifaCode: p.TeamFifaCode,
		})
	}

	if err := h.pickemService.SaveBracketPicks(r.Context(), user.ID, picks); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	pickem, err := h.pickemService.GetUserPickem(r.Context(), user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, pickem)
}
