package handlers

import (
	"net/http"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/httputils"
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
//	@Param			group_codes		query		[]string								false	"Group codes (A-L)"
//	@Param			stage_code		query		[]string								false	"Stage codes (group_stage, round_of_16, quarter_finals, semi_finals, third_place, final)"
//	@Param			status			query		string									false	"Match status (scheduled, finished)"
//	@Param			team_fifa_codes	query		[]string								false	"Team FIFA codes"
//	@Param			from_date		query		string									false	"Inclusive start date-time (RFC3339)"
//	@Param			to_date			query		string									false	"Inclusive end date-time (RFC3339)"
//	@Success		200				{object}	httputils.Response{data=[]domain.Match}	"List of matches"
//	@Failure		400				{object}	httputils.ErrorResponse					"Invalid query parameters"
//	@Failure		500				{object}	httputils.ErrorResponse					"Internal server error"
//	@Router			/matches [get]
func (h *MatchHandler) GetMatches(w http.ResponseWriter, r *http.Request) {
	groupCodes := httputils.ParseStringSliceParam(r, "group_codes")
	for i, code := range groupCodes {
		upperCode := strings.ToUpper(code)
		if !validator.IsValidGroupCode(upperCode) {
			httputils.RespondWithError(w, http.StatusBadRequest, domain.ErrInvalidGroupCode)
			return
		}
		groupCodes[i] = upperCode
	}

	stageCodes := httputils.ParseStringSliceParam(r, "stage_code")
	for _, code := range stageCodes {
		if !validator.IsValidStageCode(code) {
			httputils.RespondWithError(w, http.StatusBadRequest, domain.ErrInvalidStageCode)
			return
		}
	}

	domainStageCodes := make([]domain.MatchStageCode, len(stageCodes))
	for i, code := range stageCodes {
		domainStageCodes[i] = domain.MatchStageCode(code)
	}

	status := r.URL.Query().Get("status")
	if status != "" && !validator.IsValidStatus(status) {
		httputils.RespondWithError(w, http.StatusBadRequest, domain.ErrInvalidStatus)
		return
	}

	teamFifaCodes := httputils.ParseStringSliceParam(r, "team_fifa_codes")
	for i, code := range teamFifaCodes {
		upperCode := strings.ToUpper(code)
		if !validator.IsValidFifaCode(upperCode) {
			httputils.RespondWithError(w, http.StatusBadRequest, domain.ErrInvalidFifaCode)
			return
		}
		teamFifaCodes[i] = upperCode
	}

	fromDate, err := httputils.ParseDateParam(r, "from_date")
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err)
		return
	}

	toDate, err := httputils.ParseDateParam(r, "to_date")
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err)
		return
	}

	if fromDate != nil && toDate != nil && !validator.IsValidDateRange(fromDate, toDate) {
		httputils.RespondWithError(w, http.StatusBadRequest, domain.ErrInvalidDateRange)
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

	httputils.RespondWithData(w, http.StatusOK, matches)
}
