package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/services"
)

const playerPositionsQueryParam = "positions"
const playerTeamsQueryParam = "team_fifa_codes"

var errInvalidPlayerPosition = errors.New("invalid 'positions' value: each must be one of goalkeeper, defender, midfielder, attacker")

type PlayerHandler struct {
	playerService services.PlayerServiceInterface
	logger        logging.Logger
}

func NewPlayerHandler(playerService services.PlayerServiceInterface, logger logging.Logger) *PlayerHandler {
	return &PlayerHandler{playerService: playerService, logger: logger}
}

// SearchPlayers godoc
//
//	@Summary		Search players
//	@Description	Search the tournament player catalog. All filters are optional and combine with AND. `q` is a case-insensitive substring match across name, first name and last name. `team_fifa_codes` and `positions` accept comma-separated values (or the param repeated) and match if the player's value is in the supplied set. Results are paginated.
//	@Tags			players
//	@Produce		json
//	@Param			q					query		string				false	"Case-insensitive substring match across name, first_name and last_name"
//	@Param			team_fifa_codes		query		[]string			false	"Filter by team FIFA codes (e.g. MEX,COL). Comma-separated or repeated."	collectionFormat(csv)
//	@Param			positions			query		[]string			false	"Filter by positions. Comma-separated or repeated."	collectionFormat(csv)	Enums(goalkeeper,defender,midfielder,attacker)
//	@Param			page				query		int					false	"Page number (1-based)"	default(1)
//	@Param			limit				query		int					false	"Page size (max 100)"	default(20)
//	@Success		200					{object}	httpx.Response		"Paginated list of players"
//	@Failure		400					{object}	httpx.ErrorResponse	"Invalid query parameters"
//	@Failure		401					{object}	httpx.ErrorResponse	"Missing or invalid Bearer token"
//	@Security		BearerAuth
//	@Router			/players [get]
func (h *PlayerHandler) SearchPlayers(w http.ResponseWriter, r *http.Request) {
	page, limit := httpx.ParsePagination(w, r)
	if page == 0 {
		return
	}

	positions, err := parsePlayerPositions(r)
	if err != nil {
		httpx.BadRequest(w, r, codeInvalidQueryParam, err.Error())
		return
	}

	filters := domain.PlayerSearchFilters{
		Query:         strings.TrimSpace(r.URL.Query().Get("q")),
		TeamFifaCodes: normalizeFifaCodes(httpx.ParseStringSliceParam(r, playerTeamsQueryParam)),
		Positions:     positions,
	}

	result, err := h.playerService.SearchPlayers(r.Context(), filters, page, limit)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithPaginatedData(w, http.StatusOK, result.Players, result.Pagination)
}

func normalizeFifaCodes(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, strings.ToUpper(value))
	}
	return normalized
}

func parsePlayerPositions(r *http.Request) ([]domain.PlayerPosition, error) {
	raw := httpx.ParseStringSliceParam(r, playerPositionsQueryParam)
	if len(raw) == 0 {
		return nil, nil
	}

	positions := make([]domain.PlayerPosition, 0, len(raw))
	for _, value := range raw {
		position := domain.PlayerPosition(strings.ToLower(value))
		if !position.IsValid() {
			return nil, errInvalidPlayerPosition
		}
		positions = append(positions, position)
	}
	return positions, nil
}
