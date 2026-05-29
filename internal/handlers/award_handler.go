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

type AwardHandler struct {
	awardService services.AwardServiceInterface
	logger       logging.Logger
	validator    *validator.Validator
}

func NewAwardHandler(
	awardService services.AwardServiceInterface,
	logger logging.Logger,
	validator *validator.Validator,
) *AwardHandler {
	return &AwardHandler{awardService: awardService, logger: logger, validator: validator}
}

// GetUserAwards godoc
//
//	@Summary		Get user's award picks
//	@Description	Returns one resolved slot per award type (boot, ball, glove, young player) in canonical order, plus progress and the global lock flag.
//	@Tags			awards
//	@Produce		json
//	@Success		200	{object}	domain.UserAwards	"User's award picks state"
//	@Failure		401	{object}	httpx.ErrorResponse	"Missing or invalid Bearer token"
//	@Security		BearerAuth
//	@Router			/awards [get]
func (h *AwardHandler) GetUserAwards(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	awards, err := h.awardService.GetUserAwards(r.Context(), user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, awards)
}

// SaveAwardPicks godoc
//
//	@Summary		Save award picks
//	@Description	Replaces the user's award picks with the given set (0..4 entries, each award_type at most once). Rejected once the tournament has started, or if a picked player is ineligible for the requested award (e.g. an outfielder for Golden Glove).
//	@Tags			awards
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dtos.SaveAwardPicksDto	true	"Award picks"
//	@Success		200		{object}	domain.UserAwards		"User's award picks state"
//	@Failure		400		{object}	httpx.ErrorResponse		"Invalid request body, awards locked, or player ineligible"
//	@Failure		401		{object}	httpx.ErrorResponse		"Missing or invalid Bearer token"
//	@Security		BearerAuth
//	@Router			/awards [put]
func (h *AwardHandler) SaveAwardPicks(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	var body dtos.SaveAwardPicksDto
	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	picks := make([]*domain.UserAwardPick, 0, len(body.Picks))
	for _, pick := range body.Picks {
		picks = append(picks, &domain.UserAwardPick{
			AwardType: domain.AwardType(pick.AwardType),
			PlayerID:  pick.PlayerID,
		})
	}

	awards, err := h.awardService.SaveAwardPicks(r.Context(), user.ID, picks)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, awards)
}
