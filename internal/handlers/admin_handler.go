package handlers

import (
	"net/http"

	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/httputils"
	"github.com/fifawcp/api/internal/services"
)

type AdminHandler struct {
	matchService         services.MatchServiceInterface
	groupStandingService services.GroupStandingServiceInterface
	logger               logging.Logger
}

func NewAdminHandler(
	matchService services.MatchServiceInterface,
	groupStandingService services.GroupStandingServiceInterface,
	logger logging.Logger,
) *AdminHandler {
	return &AdminHandler{
		matchService:         matchService,
		groupStandingService: groupStandingService,
		logger:               logger,
	}
}

func (h *AdminHandler) UpdateMatchResult(w http.ResponseWriter, r *http.Request) {
	matchID := httpctx.GetMatchID(r.Context())

	var body dtos.UpdateMatchResultDto

	if err := httputils.ReadAndValidateJSON(w, r, &body, validator.NewValidator()); err != nil {
		return
	}

	outcome, err := h.matchService.UpdateMatchResult(r.Context(), matchID, body)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	// TODO: add audit log in success and error (who)
	httputils.RespondWithData(w, http.StatusOK, outcome)
}

func (h *AdminHandler) ResetMatchResult(w http.ResponseWriter, r *http.Request) {
	// TODO: Put in the document that when implementing this in the frontend
	// TODO: they should trigger a confirmation dialog, because this is a destructive action
	matchID := httpctx.GetMatchID(r.Context())

	outcome, err := h.matchService.ResetMatchResult(r.Context(), matchID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	// TODO: add audit log in success and error (who)
	httputils.RespondWithData(w, http.StatusOK, outcome)
}

func (h *AdminHandler) BulkUpdateMatchResults(w http.ResponseWriter, r *http.Request) {
	var body dtos.BulkUpdateMatchesResultDto

	if err := httputils.ReadAndValidateJSON(w, r, &body, validator.NewValidator()); err != nil {
		return
	}

	outcome, err := h.matchService.UpdateMatchResultsBulk(r.Context(), body)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	// TODO: add audit log in success and error (who)
	httputils.RespondWithData(w, http.StatusOK, outcome)
}

func (h *AdminHandler) RecalculateStandings(w http.ResponseWriter, r *http.Request) {
	outcome, err := h.matchService.SyncGroupStageOutcomes(r.Context())
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusOK, outcome)
}

// TODO: document with godoc
func (h *AdminHandler) ResolveThirdPlaceConflict(w http.ResponseWriter, r *http.Request) {
	var body dtos.ResolveThirdPlaceConflictDto

	if err := httputils.ReadAndValidateJSON(w, r, &body, validator.NewValidator()); err != nil {
		return
	}

	outcome, err := h.matchService.ResolveThirdPlaceConflict(r.Context(), body)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusOK, outcome)
}
