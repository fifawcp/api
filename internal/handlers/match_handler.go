package handlers

import (
	"net/http"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/services"
)

type MatchHandler struct {
	matchService services.MatchServiceInterface
	logger       logging.Logger
}

func NewMatchHandler(matchService services.MatchServiceInterface, logger logging.Logger) *MatchHandler {
	return &MatchHandler{
		matchService: matchService,
		logger:       logger,
	}
}

// GetMatches godoc
//
//	@Summary		List matches
//	@Description	Returns matches ordered by kickoff time ascending.
//	@Description	Supports optional filters by group code, stage code, status, team FIFA code, and kickoff date range.
//	@Description	List query params can be repeated (`?group_codes=A&group_codes=B`) or comma-separated (`?group_codes=A,B`).
//	@Tags			matches
//	@Produce		json
//	@Param			group_codes		query		[]string							false	"Group codes (A-L)"
//	@Param			stage_code		query		[]string							false	"Stage codes (group_stage, round_of_16, quarter_finals, semi_finals, third_place, final)"
//	@Param			status			query		string								false	"Match status (scheduled, finished)"
//	@Param			team_fifa_codes	query		[]string							false	"Team FIFA codes"
//	@Param			from_date		query		string								false	"Inclusive start date-time (RFC3339)"
//	@Param			to_date			query		string								false	"Inclusive end date-time (RFC3339)"
//	@Success		200				{object}	httpx.Response{data=[]domain.Match}	"List of matches"
//	@Failure		400				{object}	httpx.ErrorResponse					"Invalid query parameters"
//	@Failure		500				{object}	httpx.ErrorResponse					"Internal server error"
//	@Router			/matches [get]
func (h *MatchHandler) GetMatches(w http.ResponseWriter, r *http.Request) {
	groupCodes := httpx.ParseStringSliceParam(r, "group_codes")
	for i, code := range groupCodes {
		upperCode := strings.ToUpper(code)
		if !validator.IsValidGroupCode(upperCode) {
			// TODO: move code somewhere, and use domain error
			httpx.RespondWithError(w, r, http.StatusBadRequest, "INVALID_GROUP_CODE", "invalid group code")
			return
		}
		groupCodes[i] = upperCode
	}

	stageCodes := httpx.ParseStringSliceParam(r, "stage_code")
	for _, code := range stageCodes {
		if !validator.IsValidStageCode(code) {
			// TODO: move code somewhere, and use domain error
			httpx.RespondWithError(w, r, http.StatusBadRequest, "INVALID_STAGE_CODE", "invalid stage code")
			return
		}
	}

	domainStageCodes := make([]domain.MatchStageCode, len(stageCodes))
	for i, code := range stageCodes {
		domainStageCodes[i] = domain.MatchStageCode(code)
	}

	status := r.URL.Query().Get("status")
	if status != "" && !validator.IsValidStatus(status) {
		// TODO: move code somewhere, and use domain error
		httpx.RespondWithError(w, r, http.StatusBadRequest, "INVALID_STATUS", "invalid status")
		return
	}

	teamFifaCodes := httpx.ParseStringSliceParam(r, "team_fifa_codes")
	for i, code := range teamFifaCodes {
		upperCode := strings.ToUpper(code)
		if !validator.IsValidFifaCode(upperCode) {
			// TODO: move code somewhere, and use domain error
			httpx.RespondWithError(w, r, http.StatusBadRequest, "INVALID_FIFA_CODE", "invalid fifa code")
			return
		}
		teamFifaCodes[i] = upperCode
	}

	fromDate, err := httpx.ParseDateParam(r, "from_date")
	if err != nil {
		// TODO: move code somewhere, and use domain error
		httpx.RespondWithError(w, r, http.StatusBadRequest, "INVALID_QUERY_PARAM", err.Error())
		return
	}

	toDate, err := httpx.ParseDateParam(r, "to_date")
	if err != nil {
		// TODO: move code somewhere, and use domain error
		httpx.RespondWithError(w, r, http.StatusBadRequest, "INVALID_QUERY_PARAM", err.Error())
		return
	}

	if fromDate != nil && toDate != nil && !validator.IsValidDateRange(fromDate, toDate) {
		// TODO: move code somewhere, and use domain error
		httpx.RespondWithError(w, r, http.StatusBadRequest, "INVALID_DATE_RANGE", "from_date must be before or equal to to_date")
		return
	}

	filters := domain.MatchFilters{
		GroupCodes:    groupCodes,
		StageCodes:    domainStageCodes,
		Status:        domain.MatchStatus(status),
		TeamFifaCodes: teamFifaCodes,
		FromDate:      fromDate,
		ToDate:        toDate,
	}

	matches, err := h.matchService.GetMatches(r.Context(), filters)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, matches)
}
