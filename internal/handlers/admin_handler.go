package handlers

import (
	"fmt"
	"net/http"

	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/services"
)

type AdminHandler struct {
	matchService         services.MatchServiceInterface
	groupStandingService services.GroupStandingServiceInterface
	logger               logging.Logger
	auditLogger          logging.AuditLogger
	validator            *validator.Validator
}

func NewAdminHandler(
	matchService services.MatchServiceInterface,
	groupStandingService services.GroupStandingServiceInterface,
	logger logging.Logger,
	auditLogger logging.AuditLogger,
	validator *validator.Validator,
) *AdminHandler {
	return &AdminHandler{
		matchService:         matchService,
		groupStandingService: groupStandingService,
		logger:               logger,
		auditLogger:          auditLogger,
		validator:            validator,
	}
}

// UpdateMatchResult godoc
//
//	@Summary		Update a match result
//	@Description	Updates a single match result by match ID. Requires authentication and admin role.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Match ID"
//	@Param			body	body		dtos.UpdateMatchResultDto	true	"Match result update data"
//	@Success		200		{object}	httpx.Response				"Match result updated successfully"
//	@Failure		400		{object}	httpx.ErrorResponse			"Invalid match ID, request body, or validation error"
//	@Failure		401		{object}	httpx.ErrorResponse			"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httpx.ErrorResponse			"Forbidden - admin role required"
//	@Failure		404		{object}	httpx.ErrorResponse			"Match not found"
//	@Failure		500		{object}	httpx.ErrorResponse			"Internal server error"
//	@Security		BearerAuth
//	@Router			/admin/matches/{id}/result [post]
func (h *AdminHandler) UpdateMatchResult(w http.ResponseWriter, r *http.Request) {
	matchID := httpctx.GetMatchID(r.Context())

	var body dtos.UpdateMatchResultDto

	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	outcome, err := h.matchService.UpdateMatchResult(r.Context(), matchID, body)
	if err != nil {
		h.auditLogger.LogEvent(r.Context(), logging.Event{
			Action:     logging.ActionUpdateMatchResult,
			Resource:   logging.ResourceMatch,
			ResourceID: fmt.Sprintf("%d", matchID),
			Outcome:    logging.OutcomeFailure,
			Metadata:   map[string]any{logging.Error: err.Error()},
		})
		handleServiceError(w, r, err, h.logger)
		return
	}

	h.auditLogger.LogEvent(r.Context(), logging.Event{
		Action:     logging.ActionUpdateMatchResult,
		Resource:   logging.ResourceMatch,
		ResourceID: fmt.Sprintf("%d", matchID),
		Outcome:    logging.OutcomeSuccess,
		Metadata: map[string]any{
			"home_score": body.HomeScore,
			"away_score": body.AwayScore,
		},
	})

	httpx.RespondWithData(w, http.StatusOK, outcome)
}

// ResetMatchResult godoc
//
//	@Summary		Reset a match result
//	@Description	Resets a single match result by match ID. Requires authentication and admin role.
//	@Tags			admin
//	@Produce		json
//	@Param			id	path		string				true	"Match ID"
//	@Success		200	{object}	httpx.Response		"Match result reset successfully"
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid match ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403	{object}	httpx.ErrorResponse	"Forbidden - admin role required"
//	@Failure		404	{object}	httpx.ErrorResponse	"Match not found"
//	@Failure		500	{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/admin/matches/{id}/result [delete]
func (h *AdminHandler) ResetMatchResult(w http.ResponseWriter, r *http.Request) {
	matchID := httpctx.GetMatchID(r.Context())

	outcome, err := h.matchService.ResetMatchResult(r.Context(), matchID)
	if err != nil {
		h.auditLogger.LogEvent(r.Context(), logging.Event{
			Action:     logging.ActionResetMatchResult,
			Resource:   logging.ResourceMatch,
			ResourceID: fmt.Sprintf("%d", matchID),
			Outcome:    logging.OutcomeFailure,
			Metadata:   map[string]any{logging.Error: err.Error()},
		})
		handleServiceError(w, r, err, h.logger)
		return
	}

	h.auditLogger.LogEvent(r.Context(), logging.Event{
		Action:     logging.ActionResetMatchResult,
		Resource:   logging.ResourceMatch,
		ResourceID: fmt.Sprintf("%d", matchID),
		Outcome:    logging.OutcomeSuccess,
	})

	httpx.RespondWithData(w, http.StatusOK, outcome)
}

// BulkUpdateMatchResults godoc
//
//	@Summary		Bulk update match results
//	@Description	Updates multiple match results in a single request. Requires authentication and admin role.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dtos.BulkUpdateMatchesResultDto	true	"Bulk match result update data"
//	@Success		200		{object}	httpx.Response					"Match results updated successfully"
//	@Failure		400		{object}	httpx.ErrorResponse				"Invalid request body or validation error"
//	@Failure		401		{object}	httpx.ErrorResponse				"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httpx.ErrorResponse				"Forbidden - admin role required"
//	@Failure		500		{object}	httpx.ErrorResponse				"Internal server error"
//	@Security		BearerAuth
//	@Router			/admin/matches/results [post]
func (h *AdminHandler) BulkUpdateMatchResults(w http.ResponseWriter, r *http.Request) {
	var body dtos.BulkUpdateMatchesResultDto

	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	outcome, err := h.matchService.UpdateMatchResultsBulk(r.Context(), body)
	if err != nil {
		h.auditLogger.LogEvent(r.Context(), logging.Event{
			Action:   logging.ActionBulkUpdateMatches,
			Resource: logging.ResourceMatch,
			Outcome:  logging.OutcomeFailure,
			Metadata: map[string]any{
				logging.Error: err.Error(),
				"count":       len(body.Matches),
			},
		})
		handleServiceError(w, r, err, h.logger)
		return
	}

	h.auditLogger.LogEvent(r.Context(), logging.Event{
		Action:   logging.ActionBulkUpdateMatches,
		Resource: logging.ResourceMatch,
		Outcome:  logging.OutcomeSuccess,
		Metadata: map[string]any{
			"count": len(body.Matches),
		},
	})

	httpx.RespondWithData(w, http.StatusOK, outcome)
}

// RecalculateStandings godoc
//
//	@Summary		Recalculate group standings
//	@Description	Recalculates group standings from current match outcomes. Requires authentication and admin role.
//	@Tags			admin
//	@Produce		json
//	@Success		200	{object}	httpx.Response		"Standings recalculated successfully"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403	{object}	httpx.ErrorResponse	"Forbidden - admin role required"
//	@Failure		500	{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/admin/standings/recalculate [post]
func (h *AdminHandler) RecalculateStandings(w http.ResponseWriter, r *http.Request) {
	outcome, err := h.matchService.SyncGroupStageOutcomes(r.Context())
	if err != nil {
		h.auditLogger.LogEvent(r.Context(), logging.Event{
			Action:   logging.ActionRecalculateStandings,
			Resource: logging.ResourceStanding,
			Outcome:  logging.OutcomeFailure,
			Metadata: map[string]any{logging.Error: err.Error()},
		})
		handleServiceError(w, r, err, h.logger)
		return
	}

	h.auditLogger.LogEvent(r.Context(), logging.Event{
		Action:   logging.ActionRecalculateStandings,
		Resource: logging.ResourceStanding,
		Outcome:  logging.OutcomeSuccess,
	})

	httpx.RespondWithData(w, http.StatusOK, outcome)
}

// ResolveThirdPlaceConflict godoc
//
//	@Summary		Resolve third-place conflict
//	@Description	Resolves third-place conflicts for group standings using the provided resolution payload. Requires authentication and admin role.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dtos.ResolveThirdPlaceConflictDto	true	"Third-place conflict resolution data"
//	@Success		200		{object}	httpx.Response					"Third-place conflict resolved successfully"
//	@Failure		400		{object}	httpx.ErrorResponse				"Invalid request body or validation error"
//	@Failure		401		{object}	httpx.ErrorResponse				"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httpx.ErrorResponse				"Forbidden - admin role required"
//	@Failure		500		{object}	httpx.ErrorResponse				"Internal server error"
//	@Security		BearerAuth
//	@Router			/admin/standings/third-place/resolve [post]
func (h *AdminHandler) ResolveThirdPlaceConflict(w http.ResponseWriter, r *http.Request) {
	var body dtos.ResolveThirdPlaceConflictDto

	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	outcome, err := h.matchService.ResolveThirdPlaceConflict(r.Context(), body)
	if err != nil {
		h.auditLogger.LogEvent(r.Context(), logging.Event{
			Action:   logging.ActionResolveThirdPlace,
			Resource: logging.ResourceMatch,
			Outcome:  logging.OutcomeFailure,
			Metadata: map[string]any{
				"team_fifa_codes": body.TeamFifaCodes,
				logging.Error:     err.Error(),
			},
		})
		handleServiceError(w, r, err, h.logger)
		return
	}

	h.auditLogger.LogEvent(r.Context(), logging.Event{
		Action:   logging.ActionResolveThirdPlace,
		Resource: logging.ResourceMatch,
		Outcome:  logging.OutcomeSuccess,
		Metadata: map[string]any{
			"team_fifa_codes": body.TeamFifaCodes,
		},
	})

	httpx.RespondWithData(w, http.StatusOK, outcome)
}
