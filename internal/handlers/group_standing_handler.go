package handlers

import (
	"net/http"
	"strings"

	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/services"
)

type GroupStandingHandler struct {
	groupStandingService services.GroupStandingServiceInterface
	logger               logging.Logger
}

func NewGroupStandingHandler(
	groupStandingService services.GroupStandingServiceInterface,
	logger logging.Logger,
) *GroupStandingHandler {
	return &GroupStandingHandler{
		groupStandingService: groupStandingService,
		logger:               logger,
	}
}

// GetGroupStandings godoc
//
//	@Summary		List group standings
//	@Description	Returns group standings rows ordered by group code and position.
//	@Description	Supports optional filtering by one or more group codes and by table position.
//	@Description	List query params can be repeated (`?group_codes=A&group_codes=B`) or comma-separated (`?group_codes=A,B`).
//	@Tags			standings
//	@Produce		json
//	@Param			group_codes	query		[]string										false	"Group codes (A-L)"
//	@Param			position	query		int64											false	"Standing position to filter by"
//	@Success		200			{object}	httpx.Response{data=[]domain.GroupStanding}	"List of group standings"
//	@Failure		400			{object}	httpx.ErrorResponse							"Invalid query parameters"
//	@Failure		500			{object}	httpx.ErrorResponse							"Internal server error"
//	@Router			/standings [get]
func (h *GroupStandingHandler) GetGroupStandings(w http.ResponseWriter, r *http.Request) {
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

	// TODO: validate position 1 - 4 if provided
	position, err := httpx.ParseInt64Param(r, "position")
	if err != nil {
		httpx.RespondWithError(w, r, http.StatusBadRequest, "INVALID_QUERY_PARAM", err.Error())
		return
	}

	groupStandings, err := h.groupStandingService.GetGroupStandings(r.Context(), groupCodes, position)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, groupStandings)
}
