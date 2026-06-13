package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/services"
	"golang.org/x/sync/errgroup"
)

type MatchHandler struct {
	matchService          services.MatchServiceInterface
	matchScorePickService services.MatchScorePickServiceInterface
	logger                logging.Logger
	validator             *validator.Validator
}

func NewMatchHandler(
	matchService services.MatchServiceInterface,
	matchScorePickService services.MatchScorePickServiceInterface,
	logger logging.Logger,
	v *validator.Validator,
) *MatchHandler {
	return &MatchHandler{
		matchService:          matchService,
		matchScorePickService: matchScorePickService,
		logger:                logger,
		validator:             v,
	}
}

// GetMatches godoc
//
//	@Summary		List matches
//	@Description	Returns matches ordered by kickoff time ascending. When the caller is authenticated,
//	@Description	each match includes their score pick under `user_score_pick` (null if no pick yet)
//	@Description	Supports optional filters by group code, stage code, status, team FIFA code, and kickoff date range
//	@Description	List query params can be repeated (`?group_codes=A&group_codes=B`) or comma-separated (`?group_codes=A,B`)
//	@Tags			matches
//	@Produce		json
//	@Param			group_codes		query		[]string										false	"Group codes (A-L)"
//	@Param			stage_code		query		[]string										false	"Stage codes (group_stage, round_of_16, quarter_finals, semi_finals, third_place, final)"
//	@Param			status			query		string											false	"Match status (scheduled, finished)"
//	@Param			team_fifa_codes	query		[]string										false	"Team FIFA codes"
//	@Param			from_date		query		string											false	"Inclusive start date-time (RFC3339)"
//	@Param			to_date			query		string											false	"Inclusive end date-time (RFC3339)"
//	@Success		200				{object}	httpx.Response{data=[]dtos.MatchResponseDto}	"List of matches"
//	@Failure		400				{object}	httpx.ErrorResponse								"Invalid query parameters"
//	@Failure		500				{object}	httpx.ErrorResponse								"Internal server error"
//	@Router			/matches [get]
func (h *MatchHandler) GetMatches(w http.ResponseWriter, r *http.Request) {
	filters, err := parseMatchFilters(r)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	matches, picks, err := h.loadMatchesDataForCaller(r, filters)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, buildMatchResponse(matches, picks))
}

func (h *MatchHandler) loadMatchesDataForCaller(
	r *http.Request,
	filters domain.MatchFilters,
) ([]*domain.Match, []*domain.UserMatchScorePick, error) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	if user == nil {
		matches, err := h.matchService.GetMatches(r.Context(), filters)
		return matches, nil, err
	}

	var matches []*domain.Match
	var picks []*domain.UserMatchScorePick

	eg, egCtx := errgroup.WithContext(r.Context())

	eg.Go(func() error {
		fetched, err := h.matchService.GetMatches(egCtx, filters)
		matches = fetched
		return err
	})

	eg.Go(func() error {
		fetched, err := h.matchScorePickService.GetMatchScorePicksByUser(egCtx, user.ID)
		picks = fetched
		return err
	})

	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}

	return matches, picks, nil
}

// SaveMatchScorePick creates or updates the user's score prediction for a match.
//
//	@Summary	Save user's score pick for a match
//	@Tags		matches
//	@Accept		json
//	@Param		id		path	int							true	"Match ID"
//	@Param		body	body	dtos.SaveMatchScorePickDto	true	"Score prediction"
//	@Success	204
//	@Failure	400	{object}	httpx.ErrorResponse
//	@Failure	409	{object}	httpx.ErrorResponse	"Match already started — pick is locked"
//	@Security	BearerAuth
//	@Router		/matches/{id}/pick [put]
func (h *MatchHandler) SaveMatchScorePick(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())
	matchID := httpctx.GetMatchID(r.Context())

	var body dtos.SaveMatchScorePickDto
	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	if err := h.matchScorePickService.SaveMatchScorePick(
		r.Context(), user.ID, matchID, *body.HomeScore, *body.AwayScore,
	); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusNoContent, nil)
}

// GetMemberCompetitionPicks returns a board member's match score picks for a competition.
//
//	@Summary		Get a board member's match picks for a competition
//	@Description	Returns the locked match score picks of a specific board member for the given competition.
//	@Description	Only picks for matches that have already started (locked) are returned.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId			path		int64											true	"Board ID"
//	@Param			competitionId	path		int64											true	"Competition ID"
//	@Param			userId			path		string											true	"Member user ID (UUID)"
//	@Success		200				{object}	httpx.Response{data=[]dtos.MatchResponseDto}	"Member's match picks"
//	@Failure		400				{object}	httpx.ErrorResponse								"Competition is not match-based"
//	@Failure		401				{object}	httpx.ErrorResponse								"Missing or invalid Bearer token"
//	@Failure		404				{object}	httpx.ErrorResponse								"Board, competition, or member not found"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/competitions/{competitionId}/members/{userId}/picks [get]
func (h *MatchHandler) GetMemberCompetitionPicks(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())
	competitionID := httpctx.GetCompetitionID(r.Context())
	targetUserID := httpctx.GetUserID(r.Context())

	matches, picks, err := h.matchScorePickService.GetMemberCompetitionPicks(r.Context(), boardID, competitionID, targetUserID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, buildMatchResponse(matches, picks))
}

// GetBoardMatchPicks returns all board members' score picks for a specific match.
//
//	@Summary		Get all board members' picks for a match
//	@Description	Returns every board member alongside their predicted score for a specific match.
//	@Description	Only available after the match has started (is locked).
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path		int64											true	"Board ID"
//	@Param			id		path		int64											true	"Match ID"
//	@Success		200		{object}	httpx.Response{data=dtos.MatchMemberPicksDto}	"All members' picks for the match"
//	@Failure		401		{object}	httpx.ErrorResponse								"Missing or invalid Bearer token"
//	@Failure		403		{object}	httpx.ErrorResponse								"Match predictions not yet revealed"
//	@Failure		404		{object}	httpx.ErrorResponse								"Board or match not found"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/matches/{id}/picks [get]
func (h *MatchHandler) GetBoardMatchPicks(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())
	matchID := httpctx.GetMatchID(r.Context())

	match, memberPicks, err := h.matchScorePickService.GetBoardMatchPicks(r.Context(), boardID, matchID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, buildMatchMemberPicksResponse(match, memberPicks))
}

func buildMatchResponse(
	matches []*domain.Match,
	picks []*domain.UserMatchScorePick,
) []dtos.MatchResponseDto {
	// Index picks by match ID for O(1) lookup
	picksByMatchID := make(map[int64]*domain.UserMatchScorePick, len(picks))
	for _, pick := range picks {
		picksByMatchID[pick.MatchID] = pick
	}

	response := make([]dtos.MatchResponseDto, len(matches))
	for index, match := range matches {
		entry := dtos.MatchResponseDto{Match: match}

		// If the user has a pick for this match, add it to the response
		if pick := picksByMatchID[match.ID]; pick != nil {
			entry.UserScorePick = &dtos.UserScorePickDto{
				HomeScore: pick.HomeScore,
				AwayScore: pick.AwayScore,
			}
		}

		response[index] = entry
	}

	return response
}

func buildMatchMemberPicksResponse(
	match *domain.Match,
	memberPicks []*domain.BoardMemberMatchPick,
) dtos.MatchMemberPicksDto {
	picks := make([]dtos.MemberPickDto, len(memberPicks))
	for index, memberPick := range memberPicks {
		entry := dtos.MemberPickDto{Member: memberPick.Member}
		// HomeScore and AwayScore are always co-present (both NOT NULL in the picks table)
		if memberPick.HomeScore != nil {
			entry.Pick = &dtos.UserScorePickDto{
				HomeScore: *memberPick.HomeScore,
				AwayScore: *memberPick.AwayScore,
			}
		}
		picks[index] = entry
	}
	return dtos.MatchMemberPicksDto{Match: match, Picks: picks}
}

func parseMatchFilters(r *http.Request) (domain.MatchFilters, error) {
	groupCodes := httpx.ParseStringSliceParam(r, "group_codes")
	for index, code := range groupCodes {
		upperCode := strings.ToUpper(code)
		if !validator.IsValidGroupCode(upperCode) {
			return domain.MatchFilters{}, domain.ErrInvalidGroupCode
		}
		groupCodes[index] = upperCode
	}

	stageCodes := httpx.ParseStringSliceParam(r, "stage_code")
	for _, code := range stageCodes {
		if !validator.IsValidStageCode(code) {
			return domain.MatchFilters{}, domain.ErrInvalidStageCode
		}
	}

	domainStageCodes := make([]domain.MatchStageCode, len(stageCodes))
	for index, code := range stageCodes {
		domainStageCodes[index] = domain.MatchStageCode(code)
	}

	status := r.URL.Query().Get("status")
	if status != "" && !validator.IsValidStatus(status) {
		return domain.MatchFilters{}, domain.ErrInvalidStatus
	}

	teamFifaCodes := httpx.ParseStringSliceParam(r, "team_fifa_codes")
	for index, code := range teamFifaCodes {
		upperCode := strings.ToUpper(code)
		if !validator.IsValidFifaCode(upperCode) {
			return domain.MatchFilters{}, domain.ErrInvalidFifaCode
		}
		teamFifaCodes[index] = upperCode
	}

	fromDate, err := httpx.ParseDateParam(r, "from_date")
	if err != nil {
		return domain.MatchFilters{}, fmt.Errorf("from_date: %s: %w", err.Error(), domain.ErrInvalidQueryParam)
	}

	toDate, err := httpx.ParseDateParam(r, "to_date")
	if err != nil {
		return domain.MatchFilters{}, fmt.Errorf("to_date: %s: %w", err.Error(), domain.ErrInvalidQueryParam)
	}

	if fromDate != nil && toDate != nil && !validator.IsValidDateRange(fromDate, toDate) {
		return domain.MatchFilters{}, domain.ErrInvalidDateRange
	}

	return domain.MatchFilters{
		GroupCodes:    groupCodes,
		StageCodes:    domainStageCodes,
		Status:        domain.MatchStatus(status),
		TeamFifaCodes: teamFifaCodes,
		FromDate:      fromDate,
		ToDate:        toDate,
	}, nil
}
