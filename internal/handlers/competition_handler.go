package handlers

import (
	"net/http"
	"strings"

	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/services"
)

type CompetitionHandler struct {
	cfg                *config.Config
	validator          *validator.Validator
	logger             logging.Logger
	competitionService services.CompetitionServiceInterface
}

func NewCompetitionHandler(
	competitionService services.CompetitionServiceInterface,
	cfg *config.Config,
	validator *validator.Validator,
	logger logging.Logger,
) *CompetitionHandler {
	return &CompetitionHandler{
		competitionService: competitionService,
		cfg:                cfg,
		validator:          validator,
		logger:             logger,
	}
}

// CreateCompetition godoc
//
//	@Summary		Create a competition
//	@Description	Creates a new competition on a board. Owner or admin only. Forbidden on the global board.
//	@Tags			competitions
//	@Accept			json
//	@Produce		json
//	@Param			boardId		path		int							true	"Board ID"
//	@Param			competition	body		dtos.CreateCompetitionDto	true	"Competition data"
//	@Success		201			{object}	domain.CompetitionListItem	"Competition created"
//	@Failure		401			{object}	httpx.ErrorResponse			"Unauthorized"
//	@Failure		403			{object}	httpx.ErrorResponse			"Forbidden"
//	@Failure		404			{object}	httpx.ErrorResponse			"Board not found"
//	@Failure		409			{object}	httpx.ErrorResponse			"Pick'em or name conflict"
//	@Failure		422			{object}	httpx.ErrorResponse			"Validation error"
//	@Failure		500			{object}	httpx.ErrorResponse			"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/competitions [post]
func (h *CompetitionHandler) CreateCompetition(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())
	boardID := httpctx.GetBoardID(r.Context())
	role := httpctx.GetBoardMemberRole(r.Context())

	var body dtos.CreateCompetitionDto
	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	competition, err := h.competitionService.CreateCompetition(r.Context(), boardID, user.ID, role, body)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusCreated, competition)
}

// GetBoardCompetitions godoc
//
//	@Summary		List board competitions
//	@Description	Returns all competitions for a board with the viewer's rank and points.
//	@Tags			competitions
//	@Produce		json
//	@Param			boardId	path		int							true	"Board ID"
//	@Success		200		{array}		domain.CompetitionListItem	"Competitions"
//	@Failure		401		{object}	httpx.ErrorResponse			"Unauthorized"
//	@Failure		403		{object}	httpx.ErrorResponse			"Not a member of this board"
//	@Failure		404		{object}	httpx.ErrorResponse			"Board not found"
//	@Failure		500		{object}	httpx.ErrorResponse			"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/competitions [get]
func (h *CompetitionHandler) GetBoardCompetitions(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())
	boardID := httpctx.GetBoardID(r.Context())

	competitions, err := h.competitionService.GetBoardCompetitions(r.Context(), boardID, user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, competitions)
}

// GetLeaderboard godoc
//
//	@Summary		Get competition leaderboard
//	@Description	Returns a paginated leaderboard for a competition. Optionally filters by member username, first name, or last name.
//	@Tags			competitions
//	@Produce		json
//	@Param			competitionId	path		int					true	"Competition ID"
//	@Param			page			query		int					false	"Page number (1-indexed, default 1)"
//	@Param			limit			query		int					false	"Page size (default 20, max 100)"
//	@Param			q				query		string				false	"Filter by username, first name, or last name (case-insensitive substring match)"
//	@Success		200				{object}	httpx.Response		"Leaderboard page retrieved successfully"
//	@Failure		400				{object}	httpx.ErrorResponse	"Invalid pagination params"
//	@Failure		401				{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		404				{object}	httpx.ErrorResponse	"Competition not found"
//	@Failure		500				{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/competitions/{competitionId}/leaderboard [get]
func (h *CompetitionHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	competitionID := httpctx.GetCompetitionID(r.Context())
	page, limit := httpx.ParsePagination(w, r)

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 64 {
		q = q[:64]
	}

	sort := r.URL.Query().Get("sort")
	dir := r.URL.Query().Get("dir")

	leaderboardPage, err := h.competitionService.GetLeaderboard(r.Context(), competitionID, page, limit, q, sort, dir)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithPaginatedData(w, http.StatusOK, leaderboardPage.Members, leaderboardPage.Pagination)
}

// GetBoardSummary godoc
//
//	@Summary		Get board summary standings
//	@Description	Per-member points across all the board's competitions (per-type subtotals + raw-sum total), ranked.
//	@Tags			competitions
//	@Produce		json
//	@Param			boardId	path		int					true	"Board ID"
//	@Param			page	query		int					false	"Page number (1-indexed, default 1)"
//	@Param			limit	query		int					false	"Page size (default 20, max 100)"
//	@Param			q		query		string				false	"Filter by username, first name, or last name"
//	@Success		200		{object}	httpx.Response		"Summary page retrieved successfully"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/summary [get]
func (h *CompetitionHandler) GetBoardSummary(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())
	page, limit := httpx.ParsePagination(w, r)

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 64 {
		q = q[:64]
	}

	sort := r.URL.Query().Get("sort")
	dir := r.URL.Query().Get("dir")

	summary, err := h.competitionService.GetBoardSummary(r.Context(), boardID, page, limit, q, sort, dir)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithPaginatedData(w, http.StatusOK, summary.Members, summary.Pagination)
}

// DeleteCompetition godoc
//
//	@Summary		Delete a competition
//	@Description	Deletes a competition. Owner or admin only. Forbidden on global board competitions.
//	@Tags			competitions
//	@Produce		json
//	@Param			competitionId	path	int	true	"Competition ID"
//	@Success		204				"Competition deleted"
//	@Failure		401				{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403				{object}	httpx.ErrorResponse	"Forbidden"
//	@Failure		404				{object}	httpx.ErrorResponse	"Competition not found"
//	@Failure		500				{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/competitions/{competitionId} [delete]
func (h *CompetitionHandler) DeleteCompetition(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())
	competitionID := httpctx.GetCompetitionID(r.Context())
	role := httpctx.GetBoardMemberRole(r.Context())

	if err := h.competitionService.DeleteCompetition(r.Context(), boardID, competitionID, role); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusNoContent, nil)
}
