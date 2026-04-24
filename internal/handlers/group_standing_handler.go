package handlers

import (
	"net/http"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/httputils"
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
//	@Success		200			{object}	httputils.Response{data=[]domain.GroupStanding}	"List of group standings"
//	@Failure		400			{object}	httputils.ErrorResponse							"Invalid query parameters"
//	@Failure		500			{object}	httputils.ErrorResponse							"Internal server error"
//	@Router			/standings [get]
func (h *GroupStandingHandler) GetGroupStandings(w http.ResponseWriter, r *http.Request) {
	groupCodes := httputils.ParseStringSliceParam(r, "group_codes")

	for i, code := range groupCodes {
		upperCode := strings.ToUpper(code)

		if !validator.IsValidGroupCode(upperCode) {
			httputils.RespondWithError(w, http.StatusBadRequest, domain.ErrInvalidGroupCode)
			return
		}
		groupCodes[i] = upperCode
	}

	// TODO: validate position 1 - 4 if provided
	position, err := httputils.ParseInt64Param(r, "position")
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err)
		return
	}

	groupStandings, err := h.groupStandingService.GetGroupStandings(r.Context(), groupCodes, position)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusOK, groupStandings)
}
