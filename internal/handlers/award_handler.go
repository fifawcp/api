package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/services"
)

const (
	popularLimitDefault = 10
	popularLimitMax     = 50
)

var errPopularLimitOutOfRange = errors.New("limit must be between 1 and 50")

func parsePopularLimit(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return popularLimitDefault, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 1 || parsed > popularLimitMax {
		return 0, errPopularLimitOutOfRange
	}
	return parsed, nil
}

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

// GetPopularPicks godoc
//
//	@Summary		Get popular award picks
//	@Description	Returns the most-picked players per award (across all users), ranked desc by pick count then alphabetically. Eligibility is enforced per award: Golden Glove restricts to goalkeepers; Young Player restricts to players within the age cap (unknown ages are treated as eligible). Boot and Ball draw from the full catalog. Useful for the awards picker UI's "top choices" section.
//	@Tags			awards
//	@Produce		json
//	@Param			limit	query		int							false	"Top-N per award (1-50)"	default(10)
//	@Success		200		{object}	domain.PopularPicksByAward	"Top picks grouped by award type"
//	@Failure		400		{object}	httpx.ErrorResponse			"Invalid limit"
//	@Failure		401		{object}	httpx.ErrorResponse			"Missing or invalid Bearer token"
//	@Security		BearerAuth
//	@Router			/awards/popular [get]
func (h *AwardHandler) GetPopularPicks(w http.ResponseWriter, r *http.Request) {
	limit, err := parsePopularLimit(r)
	if err != nil {
		httpx.BadRequest(w, r, codeInvalidQueryParam, err.Error())
		return
	}

	picks, err := h.awardService.GetPopularPicks(r.Context(), limit)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, picks)
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
