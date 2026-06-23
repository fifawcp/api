package handlers

import (
	"net/http"

	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/services"
)

type DashboardHandler struct {
	dashboardService services.DashboardServiceInterface
	logger           logging.Logger
}

func NewDashboardHandler(
	dashboardService services.DashboardServiceInterface,
	logger logging.Logger,
) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
		logger:           logger,
	}
}

// GetDashboard godoc
//
//	@Summary		Dashboard for authenticated or guest callers
//	@Description	Returns the next scheduled matches (all that kick off at the same earliest time, e.g. simultaneous group finales) and the top 5 of each global competition for everyone.
//	@Description	`next_match` is the earliest of those and is deprecated in favour of `next_matches`.
//	@Description	When the caller is authenticated, also includes their picked champion (when bracket is complete),
//	@Description	per-competition rank and points, and pick progress (match picks + pickem groups/best-thirds/bracket).
//	@Description	Guest callers receive `null` for `picked_champion`, `stats`, and `progress`.
//	@Tags			dashboard
//	@Produce		json
//	@Success		200	{object}	httpx.Response{data=dtos.DashboardResponseDto}	"Dashboard"
//	@Failure		500	{object}	httpx.ErrorResponse								"Internal server error"
//	@Router			/dashboard [get]
func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	var userID string
	if user != nil {
		userID = user.ID
	}

	dashboard, err := h.dashboardService.GetDashboard(r.Context(), userID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, dtos.NewDashboardResponse(dashboard))
}
