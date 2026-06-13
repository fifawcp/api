package main

import (
	"context"
	"database/sql"
	"fmt"
	mathrand "math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/services"
)

type Seeder struct {
	db                    *sql.DB
	logger                logging.Logger
	userRepository        domain.UserRepository
	boardRepository       domain.BoardRepository
	boardMemberRepository domain.BoardMemberRepository
	pickemRepository      domain.PickemRepository
	matchRepository       domain.MatchRepository
	awardPickRepository   domain.AwardPickRepository
	matchService          services.MatchServiceInterface
	pickemService         services.PickemServiceInterface
	boardService          services.BoardServiceInterface
	competitionService    services.CompetitionServiceInterface
	boardOwners           map[int64]string
}

func NewSeeder(
	db *sql.DB,
	logger logging.Logger,
	userRepository domain.UserRepository,
	boardRepository domain.BoardRepository,
	boardMemberRepository domain.BoardMemberRepository,
	pickemRepository domain.PickemRepository,
	matchRepository domain.MatchRepository,
	awardPickRepository domain.AwardPickRepository,
	matchService services.MatchServiceInterface,
	pickemService services.PickemServiceInterface,
	boardService services.BoardServiceInterface,
	competitionService services.CompetitionServiceInterface,
) *Seeder {
	return &Seeder{
		db:                    db,
		logger:                logger,
		userRepository:        userRepository,
		boardRepository:       boardRepository,
		boardMemberRepository: boardMemberRepository,
		pickemRepository:      pickemRepository,
		matchRepository:       matchRepository,
		awardPickRepository:   awardPickRepository,
		matchService:          matchService,
		pickemService:         pickemService,
		boardService:          boardService,
		competitionService:    competitionService,
		boardOwners:           map[int64]string{},
	}
}

func (s *Seeder) Flush() {
	s.logger.Info("Flushing database")
	ctx := context.Background()

	queries := []string{
		"DELETE FROM boards WHERE name != 'Global';", // cascades to board_members, competitions, scopes, scores
		"DELETE FROM users;",                         // cascades to picks, score_events, board_members
	}
	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			s.logger.Error("Error flushing db", logging.Error, err.Error())
		}
	}

	if err := s.resetTournamentState(ctx); err != nil {
		s.logger.Error("Error resetting tournament state", logging.Error, err.Error())
	}

	if err := s.resetMatchesToCanonical(ctx); err != nil {
		s.logger.Error("Error resetting matches", logging.Error, err.Error())
	}
}

func (s *Seeder) generateUsers(amount int) []*domain.User {
	users := make([]*domain.User, amount)

	for i := range amount {
		firstName := gofakeit.FirstName()
		lastName := gofakeit.LastName()
		indexSuffix := strconv.Itoa(i)

		firstNameLower := strings.ToLower(firstName)
		maxFirstNameLength := usernameMaxLength - len(indexSuffix)
		if len(firstNameLower) > maxFirstNameLength {
			firstNameLower = firstNameLower[:maxFirstNameLength]
		}

		username := firstNameLower + indexSuffix
		email := username + "@email.com"

		users[i] = &domain.User{
			FirstName: firstName,
			LastName:  lastName,
			Username:  username,
			Email:     email,
		}
	}

	return users
}

func (s *Seeder) seedBoards(ctx context.Context, users []*domain.User) {
	for i := range boardsAmount {
		owner := users[i]
		ownerID := owner.ID

		name := boardNames[mathrand.Intn(len(boardNames))]
		if len(name) > boardNameMaxLength {
			name = name[:boardNameMaxLength]
		}

		board, err := s.boardService.CreateBoard(ctx, dtos.CreateBoardDto{Name: name}, ownerID)
		if err != nil {
			s.logger.Error(
				"Error seeding board",
				logging.Error, err.Error(),
			)
			continue
		}
		s.boardOwners[board.ID] = ownerID

		memberCount := boardMembersMin + mathrand.Intn(boardMembersMax-boardMembersMin+1)
		candidates := make([]*domain.User, 0, len(users)-1)

		for _, user := range users {
			if user.ID != owner.ID {
				candidates = append(candidates, user)
			}
		}

		mathrand.Shuffle(len(candidates), func(a, b int) {
			candidates[a], candidates[b] = candidates[b], candidates[a]
		})

		for j := 0; j < memberCount && j < len(candidates); j++ {
			if _, err := s.boardMemberRepository.CreateBoardMember(ctx, *board.JoinCode, candidates[j].ID); err != nil {
				s.logger.Error(
					"Error seeding board member",
					logging.Error, err.Error(),
				)
			}
		}
	}
}

func (s *Seeder) seedPickemData(ctx context.Context, users []*domain.User) {
	groupCodes := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}

	for _, user := range users {
		picks := make([]*domain.UserGroupPick, 0, 48)
		thirdPlaceTeams := make([]string, 0, 12)

		for _, groupCode := range groupCodes {
			teams := make([]string, len(teamsByGroup[groupCode]))
			copy(teams, teamsByGroup[groupCode])
			mathrand.Shuffle(len(teams), func(a, b int) { teams[a], teams[b] = teams[b], teams[a] })

			for pos, fifaCode := range teams {
				picks = append(picks, &domain.UserGroupPick{
					UserID:            user.ID,
					TeamFifaCode:      fifaCode,
					TeamGroupCode:     groupCode,
					PredictedPosition: pos + 1,
				})

				if pos == 2 {
					thirdPlaceTeams = append(thirdPlaceTeams, fifaCode)
				}
			}
		}

		if err := s.pickemRepository.UpsertGroupPicks(ctx, user.ID, picks); err != nil {
			s.logger.Error("Error seeding group picks", logging.Error, err.Error())
			continue
		}

		// Seeded pickem users have committed their predictions, so every group ships pre-locked
		lockPlaceholders := make([]string, 0, len(groupCodes))
		lockArgs := make([]any, 0, len(groupCodes)*2)
		for i, code := range groupCodes {
			base := i * 2
			lockPlaceholders = append(lockPlaceholders, fmt.Sprintf("($%d, $%d)", base+1, base+2))
			lockArgs = append(lockArgs, user.ID, code)
		}
		lockQuery := `INSERT INTO user_group_locks (user_id, group_code) VALUES ` +
			strings.Join(lockPlaceholders, ", ") + ` ON CONFLICT DO NOTHING`
		if _, err := s.db.ExecContext(ctx, lockQuery, lockArgs...); err != nil {
			s.logger.Error("Error seeding group locks", logging.Error, err.Error())
		}

		mathrand.Shuffle(len(thirdPlaceTeams), func(a, b int) {
			thirdPlaceTeams[a], thirdPlaceTeams[b] = thirdPlaceTeams[b], thirdPlaceTeams[a]
		})

		bestThirds := make([]*domain.UserBestThirdPick, 8)
		for i, code := range thirdPlaceTeams[:8] {
			bestThirds[i] = &domain.UserBestThirdPick{
				UserID:       user.ID,
				TeamFifaCode: code,
			}
		}

		if err := s.pickemRepository.UpsertBestThirds(ctx, user.ID, bestThirds); err != nil {
			s.logger.Error("Error seeding best thirds", logging.Error, err.Error())
		}
	}
}

func (s *Seeder) seedMatchScoresFor(ctx context.Context, users []*domain.User, matchIDs []int64) {
	if len(matchIDs) == 0 || len(users) == 0 {
		return
	}

	placeholders := make([]string, len(matchIDs))
	for i := range matchIDs {
		base := i * 4
		placeholders[i] = fmt.Sprintf("($%d::uuid, $%d::bigint, $%d::int, $%d::int)", base+1, base+2, base+3, base+4)
	}

	query := `
		INSERT INTO user_match_score_picks (user_id, match_id, home_score, away_score)
		SELECT
			v.user_id,
			v.match_id,
			v.home_score,
			v.away_score
		FROM (VALUES ` + strings.Join(placeholders, ", ") + `)
		  AS v(user_id, match_id, home_score, away_score)
		INNER JOIN matches m
		  ON m.id = v.match_id
		 	AND m.home_team_fifa_code IS NOT NULL
		 	AND m.away_team_fifa_code IS NOT NULL
		ON CONFLICT (user_id, match_id) DO NOTHING`

	for _, user := range users {
		args := make([]any, 0, len(matchIDs)*4)

		for _, matchID := range matchIDs {
			args = append(args, user.ID, matchID, mathrand.Intn(6), mathrand.Intn(6))
		}

		if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
			s.logger.Error("Error seeding match scores", logging.Error, err.Error())
		}
	}
}

// RunScenario wipes the DB back to a known canonical state
// and reconstructs a tournament snapshot at the given milestone
func (s *Seeder) RunScenario(ctx context.Context, scenarioName string) error {
	scenario, err := resolveScenario(scenarioName)
	if err != nil {
		return err
	}
	s.logger.Info("Running scenario", "scenario", scenario.Name)

	s.Flush()

	if err := s.shiftKickoffDates(ctx, scenario); err != nil {
		return fmt.Errorf("shift kickoff dates: %w", err)
	}

	users := s.generateUsers(usersAmount)
	createdUsers := make([]*domain.User, 0, len(users))
	for _, user := range users {
		if err := s.userRepository.CreateUser(ctx, user); err != nil {
			s.logger.Error("Error seeding users", logging.Error, err.Error())
			continue
		}

		createdUsers = append(createdUsers, user)
	}

	s.seedBoards(ctx, createdUsers)
	if err := s.seedCompetitions(ctx); err != nil {
		return fmt.Errorf("seed competitions: %w", err)
	}

	pickemUsers, matchUsers, err := partitionUsersByEngagement(createdUsers)
	if err != nil {
		return err
	}

	s.seedPickemData(ctx, pickemUsers)
	if err := s.seedBracketPicks(ctx, pickemUsers); err != nil {
		return fmt.Errorf("seed bracket picks: %w", err)
	}
	if err := s.seedAwardPicks(ctx, pickemUsers); err != nil {
		return fmt.Errorf("seed award picks: %w", err)
	}

	// Group-stage teams are always assigned post-migration, so these picks always land
	// Knockout picks are seeded per-stage below, once their matchups become known via advanceBracket
	s.seedMatchScoresFor(ctx, matchUsers, allMatchIDs())

	scriptedResultsByMatchID, err := buildScriptedResults(ctx, s.db)
	if err != nil {
		return fmt.Errorf("build scripted results: %w", err)
	}

	for stageIndex, stageMatchIDs := range scenario.StageGroups {
		if err := s.applyScenarioResults(ctx, stageMatchIDs, scriptedResultsByMatchID); err != nil {
			return fmt.Errorf("apply stage %d results: %w", stageIndex, err)
		}

		// After this stage's apply, advanceBracket has filled in the NEXT
		// stage's home/away teams. Re-seed picks so that next stage's
		// scoring (in the following iteration) sees them
		s.seedMatchScoresFor(ctx, matchUsers, allMatchIDs())
	}

	s.logger.Info("Scenario seeded successfully", "scenario", scenario.Name)
	return nil
}

func partitionUsersByEngagement(users []*domain.User) (pickemUsers, matchUsers []*domain.User, err error) {
	sum := userEngagement.PickemAndMatch + userEngagement.PickemOnly + userEngagement.MatchOnly + userEngagement.Idle
	if delta := sum - 1.0; delta > 0.001 || delta < -0.001 {
		return nil, nil, fmt.Errorf(
			"userEngagement buckets sum to %.3f (want 1.0): pickem+match=%.2f, pickemOnly=%.2f, matchOnly=%.2f, idle=%.2f",
			sum, userEngagement.PickemAndMatch, userEngagement.PickemOnly, userEngagement.MatchOnly, userEngagement.Idle,
		)
	}

	mathrand.Shuffle(len(users), func(a, b int) {
		users[a], users[b] = users[b], users[a]
	})

	userCount := len(users)
	bothEnd := int(float64(userCount) * userEngagement.PickemAndMatch)
	pickemOnlyEnd := bothEnd + int(float64(userCount)*userEngagement.PickemOnly)
	matchOnlyEnd := pickemOnlyEnd + int(float64(userCount)*userEngagement.MatchOnly)

	both := users[:bothEnd]
	pickemOnly := users[bothEnd:pickemOnlyEnd]
	matchOnly := users[pickemOnlyEnd:matchOnlyEnd]

	pickemUsers = append(pickemUsers, both...)
	pickemUsers = append(pickemUsers, pickemOnly...)
	matchUsers = append(matchUsers, both...)
	matchUsers = append(matchUsers, matchOnly...)
	return pickemUsers, matchUsers, nil
}

func allMatchIDs() []int64 {
	ids := make([]int64, 104)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	return ids
}

// resetTournamentState clears every piece of derived/user-generated state that downstream re-seeding will reconstruct
func (s *Seeder) resetTournamentState(ctx context.Context) error {
	queries := []string{
		`DELETE FROM score_events`,
		`DELETE FROM competition_match_scores`,
		`DELETE FROM user_bracket_picks`,
		`DELETE FROM user_match_score_picks`,
		`DELETE FROM user_award_picks`,
		`DELETE FROM competition_scope_stages WHERE competition_id IN (SELECT id FROM competitions WHERE board_id NOT IN (SELECT id FROM boards WHERE privacy = 'global'))`,
		`DELETE FROM competition_scope_teams  WHERE competition_id IN (SELECT id FROM competitions WHERE board_id NOT IN (SELECT id FROM boards WHERE privacy = 'global'))`,
		`DELETE FROM competitions WHERE board_id NOT IN (SELECT id FROM boards WHERE privacy = 'global')`,
		`UPDATE group_standings SET position = 0, matches_played = 0, wins = 0, draws = 0, losses = 0, goals_for = 0, goals_against = 0, goal_difference = 0, points = 0, updated_at = NOW()`,
	}

	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("%s: %w", query, err)
		}
	}

	return nil
}

// resetMatchesToCanonical clears every match's result fields back to the scheduled state and re-NULLs any bracket team assignments that prior scenario runs (promoteGroupWinners + advanceBracket) may have written
func (s *Seeder) resetMatchesToCanonical(ctx context.Context) error {
	queries := []string{
		`UPDATE matches SET
			home_score = NULL,
			away_score = NULL,
			home_penalty_score = NULL,
			away_penalty_score = NULL,
			winner_team_fifa_code = NULL,
			status = 'scheduled',
			updated_at = NOW()`,
		`UPDATE matches SET
			home_team_fifa_code = NULL,
			away_team_fifa_code = NULL,
			updated_at = NOW()
		WHERE stage_code != 'group_stage'`,
	}

	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("%s: %w", query, err)
		}
	}

	return nil
}

// shiftKickoffDates moves every match's kickoff_at by a single offset computed
// so the scenario's anchor match sits at its target time. The shift is uniform,
// so anchoring one match re-pins the whole schedule while preserving spacing.
// Normal scenarios anchor to "now + AnchorOffset"; real-dates scenarios anchor
// the opener to its canonical absolute kickoff, restoring the real 2026 schedule.
func (s *Seeder) shiftKickoffDates(ctx context.Context, scenario Scenario) error {
	var currentAnchorKickoff time.Time
	row := s.db.QueryRowContext(ctx, `SELECT kickoff_at FROM matches WHERE id = $1`, scenario.AnchorMatchID)
	if err := row.Scan(&currentAnchorKickoff); err != nil {
		return fmt.Errorf("read anchor kickoff: %w", err)
	}

	targetAnchorKickoff := time.Now().UTC().Add(scenario.AnchorOffset)
	if scenario.UseRealDates {
		targetAnchorKickoff = canonicalOpenerKickoff
	}
	offset := targetAnchorKickoff.Sub(currentAnchorKickoff)
	intervalSeconds := int64(offset.Seconds())
	if _, err := s.db.ExecContext(ctx,
		`UPDATE matches SET kickoff_at = kickoff_at + make_interval(secs => $1::double precision)`,
		intervalSeconds,
	); err != nil {
		return fmt.Errorf("apply kickoff offset: %w", err)
	}

	s.logger.Info("Shifted kickoff dates",
		"anchor_match", scenario.AnchorMatchID,
		"offset_hours", offset.Hours(),
	)

	return nil
}

// seedCompetitions creates 1-3 random custom (match) competitions plus 1-2
// single-match (pick) competitions on every non-Global seeded board, drawing
// from competitionTemplates and pickCompetitionTemplates respectively
func (s *Seeder) seedCompetitions(ctx context.Context) error {
	for boardID, ownerID := range s.boardOwners {
		matchCount := 1 + mathrand.Intn(3)
		s.seedCompetitionsFrom(ctx, boardID, ownerID, competitionTemplates, matchCount)

		pickCount := 1 + mathrand.Intn(2)
		s.seedCompetitionsFrom(ctx, boardID, ownerID, pickCompetitionTemplates, pickCount)
	}

	return nil
}

// seedCompetitionsFrom creates `count` competitions on a board, drawn at random
// (no repeats) from the given templates.
func (s *Seeder) seedCompetitionsFrom(ctx context.Context, boardID int64, ownerID string, templates []dtos.CreateCompetitionDto, count int) {
	shuffled := make([]dtos.CreateCompetitionDto, len(templates))
	copy(shuffled, templates)
	mathrand.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })

	for i := 0; i < count && i < len(shuffled); i++ {
		template := shuffled[i]

		if _, err := s.competitionService.CreateCompetition(ctx, boardID, ownerID, domain.BoardMemberRoleOwner, template); err != nil {
			s.logger.Error("Error seeding competition",
				logging.Error, err.Error(),
				"board_id", boardID,
				"name", template.Name,
			)
		}
	}
}

var knockoutStages = [][]int64{
	{73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88}, // round of 32
	{89, 90, 91, 92, 93, 94, 95, 96},                                 // round of 16
	{97, 98, 99, 100},                                                // quarter-finals
	{101, 102},                                                       // semi-finals
	{103, 104},                                                       // third place + final
}

// seedBracketPicks generates a bracket pick for every knockout match for each
// pickem user. Picks are constructed stage-by-stage from the live projection
// returned by PickemService — each pick is always one of the two teams the
// projection shows in the slot, so downstream matches resolve cleanly
func (s *Seeder) seedBracketPicks(ctx context.Context, users []*domain.User) error {
	for _, user := range users {
		accumulatedPicks := make([]*domain.UserBracketPick, 0, 32)

		for _, stageMatchIDs := range knockoutStages {
			pickem, err := s.pickemService.GetUserPickem(ctx, user.ID)
			if err != nil {
				s.logger.Error("Error projecting bracket for seed", logging.Error, err.Error(), "user_id", user.ID)
				break
			}

			slotByMatchID := make(map[int64]*domain.BracketMatchSlot, len(pickem.Bracket))
			for _, slot := range pickem.Bracket {
				slotByMatchID[slot.MatchID] = slot
			}

			for _, matchID := range stageMatchIDs {
				slot, ok := slotByMatchID[matchID]
				if !ok || slot.HomeTeam == nil || slot.AwayTeam == nil {
					continue
				}

				pickedTeam := slot.HomeTeam
				if mathrand.Intn(2) == 1 {
					pickedTeam = slot.AwayTeam
				}

				accumulatedPicks = append(accumulatedPicks, &domain.UserBracketPick{
					UserID:       user.ID,
					MatchID:      matchID,
					TeamFifaCode: pickedTeam.FifaCode,
				})
			}

			if err := s.pickemRepository.UpsertBracketPicks(ctx, user.ID, accumulatedPicks); err != nil {
				s.logger.Error("Error seeding bracket picks", logging.Error, err.Error(), "user_id", user.ID)
				break
			}
		}
	}

	return nil
}

// applyScenarioResults submits scripted results for the given match IDs via
// MatchService.UpdateMatchResultsBulkSync, then propagates winners into
// downstream knockout slots via advanceBracket. The match-service call
// internally handles group-stage promotion + the scoring fan-out.
func (s *Seeder) applyScenarioResults(ctx context.Context, matchIDs []int64, results map[int64]ScriptedResult) error {
	if len(matchIDs) == 0 {
		return nil
	}

	payload := dtos.BulkUpdateMatchesResultDto{
		Matches: make([]dtos.BulkUpdateMatchResultDto, 0, len(matchIDs)),
	}

	for _, matchID := range matchIDs {
		result, ok := results[matchID]
		if !ok {
			return fmt.Errorf("no scripted result for match %d", matchID)
		}

		homeScore := result.HomeScore
		awayScore := result.AwayScore
		payload.Matches = append(payload.Matches, dtos.BulkUpdateMatchResultDto{
			ID: matchID,
			UpdateMatchResultDto: dtos.UpdateMatchResultDto{
				HomeScore:        &homeScore,
				AwayScore:        &awayScore,
				HomePenaltyScore: result.HomePenaltyScore,
				AwayPenaltyScore: result.AwayPenaltyScore,
			},
		})
	}

	if _, err := s.matchService.UpdateMatchResultsBulkSync(ctx, payload); err != nil {
		return fmt.Errorf("update match results: %w", err)
	}

	if err := s.advanceBracket(ctx, matchIDs); err != nil {
		return fmt.Errorf("advance bracket: %w", err)
	}

	return nil
}

// advanceBracket fills downstream knockout slots based on the winners/losers
// of just-finished matches. MatchService.SyncGroupStageOutcomes already
// handles group → R32 promotion; this covers R32 → R16, R16 → QF, QF → SF,
// and SF → 3rd-place/Final, which aren't wired in the production service
func (s *Seeder) advanceBracket(ctx context.Context, completedMatchIDs []int64) error {
	if len(completedMatchIDs) == 0 {
		return nil
	}

	finishedMatches, err := s.matchRepository.GetMatches(ctx, domain.MatchFilters{MatchIDs: completedMatchIDs})
	if err != nil {
		return err
	}

	completed := map[int64]*domain.Match{}
	for _, match := range finishedMatches {
		completed[match.ID] = match
	}

	var updates []domain.MatchTeamUpdate
	for downstreamID, rule := range domain.MatchSlotRules {
		var update domain.MatchTeamUpdate
		update.MatchID = downstreamID
		anySet := false

		if homeCode := resolveBracketSource(rule.Home, completed); homeCode != "" {
			homeCodeCopy := homeCode
			update.HomeTeamFifaCode = &homeCodeCopy
			anySet = true
		}

		if awayCode := resolveBracketSource(rule.Away, completed); awayCode != "" {
			awayCodeCopy := awayCode
			update.AwayTeamFifaCode = &awayCodeCopy
			anySet = true
		}

		if anySet {
			updates = append(updates, update)
		}
	}

	if len(updates) == 0 {
		return nil
	}

	sort.Slice(updates, func(a, b int) bool { return updates[a].MatchID < updates[b].MatchID })
	return s.matchRepository.UpdateMatchTeams(ctx, updates)
}

func resolveBracketSource(src domain.Source, completed map[int64]*domain.Match) string {
	if src.Kind != domain.SourceWinner && src.Kind != domain.SourceLoser {
		return ""
	}

	match, ok := completed[src.MatchID]
	if !ok || match.Result == nil || match.Result.WinnerTeamFifaCode == nil {
		return ""
	}

	winner := *match.Result.WinnerTeamFifaCode
	if src.Kind == domain.SourceWinner {
		return winner
	}

	if match.Teams.Home != nil && match.Teams.Home.FifaCode == winner && match.Teams.Away != nil {
		return match.Teams.Away.FifaCode
	}

	if match.Teams.Home != nil {
		return match.Teams.Home.FifaCode
	}

	return ""
}
