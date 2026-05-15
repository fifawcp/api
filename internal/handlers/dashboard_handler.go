package handlers

import (
	"net/http"

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
//	@Summary		Authenticated user's full dashboard
//	@Description	Returns all dashboard data in a single call: picked champion (only when bracket is complete),
//	@Description	per-competition rank and points on the global board's pickem and match competitions,
//	@Description	next scheduled match, match-picks completion count, and the top 5 of each global competition.
//	@Tags			dashboard
//	@Produce		json
//	@Success		200	{object}	httpx.Response{data=domain.Dashboard}	"Dashboard"
//	@Failure		401	{object}	httpx.ErrorResponse						"Missing or invalid Bearer token"
//	@Failure		500	{object}	httpx.ErrorResponse						"Internal server error"
//	@Security		BearerAuth
//	@Router			/dashboard [get]
func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	dashboard, err := h.dashboardService.GetDashboard(r.Context(), user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, dashboard)
}
