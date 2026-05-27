package services

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

//go:embed data/combinations.json
var combinationsJSON []byte

type MatchServiceInterface interface {
	GetMatches(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error)
	UpdateMatchResult(ctx context.Context, matchID int64, payload dtos.UpdateMatchResultDto) (*domain.SyncGroupStageOutcomes, error)
	UpdateMatchResultsBulk(ctx context.Context, payload dtos.BulkUpdateMatchesResultDto) (*domain.SyncGroupStageOutcomes, error)
	UpdateMatchResultsBulkSync(ctx context.Context, payload dtos.BulkUpdateMatchesResultDto) (*domain.SyncGroupStageOutcomes, error)
	ResetMatchResult(ctx context.Context, matchID int64) (*domain.SyncGroupStageOutcomes, error)
	SyncGroupStageOutcomes(ctx context.Context) (*domain.SyncGroupStageOutcomes, error)
	ResolveThirdPlaceConflict(ctx context.Context, payload dtos.ResolveThirdPlaceConflictDto) (*domain.SyncGroupStageOutcomes, error)
}

type MatchService struct {
	matchRepository           domain.MatchRepository
	groupStandingRepository   domain.GroupStandingRepository
	groupStandingService      GroupStandingServiceInterface
	scoringService            ScoringServiceInterface
	competitionScoringService CompetitionScoringServiceInterface
	logger                    logging.Logger
	combinations              []domain.ThirdPlaceCombination
}

func NewMatchService(
	matchRepository domain.MatchRepository,
	groupStandingRepository domain.GroupStandingRepository,
	groupStandingService GroupStandingServiceInterface,
	scoringService ScoringServiceInterface,
	competitionScoringService CompetitionScoringServiceInterface,
	logger logging.Logger,
) *MatchService {
	return &MatchService{
		matchRepository:           matchRepository,
		groupStandingRepository:   groupStandingRepository,
		groupStandingService:      groupStandingService,
		scoringService:            scoringService,
		competitionScoringService: competitionScoringService,
		logger:                    logger,
		combinations:              loadThirdPlaceCombinations(),
	}
}

func (s *MatchService) GetMatches(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
	return s.matchRepository.GetMatches(ctx, filters)
}

func (s *MatchService) UpdateMatchResult(ctx context.Context, matchID int64, payload dtos.UpdateMatchResultDto) (*domain.SyncGroupStageOutcomes, error) {
	matches, err := s.matchRepository.GetMatches(ctx, domain.MatchFilters{MatchIDs: []int64{matchID}})
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, domain.ErrMatchNotFound
	}

	if matches[0].Teams.Home == nil || matches[0].Teams.Away == nil {
		return nil, domain.ErrMatchTeamsNotAssigned
	}

	if err := validateMatchResultForStage(matches[0].StageCode, payload); err != nil {
		return nil, err
	}

	updatedMatch := domain.MatchResultUpdate{
		MatchID:          matchID,
		HomeScore:        *payload.HomeScore,
		AwayScore:        *payload.AwayScore,
		HomePenaltyScore: payload.HomePenaltyScore,
		AwayPenaltyScore: payload.AwayPenaltyScore,
		Status:           domain.MatchStatusFinished,
	}
	if err := s.matchRepository.UpdateMatchesResult(ctx, []domain.MatchResultUpdate{updatedMatch}); err != nil {
		return nil, err
	}

	outcomes, err := s.SyncGroupStageOutcomes(ctx)
	if err != nil {
		return nil, err
	}

	s.asyncScoreMatches(outcomes, []int64{matchID})
	return outcomes, nil
}

func (s *MatchService) UpdateMatchResultsBulk(ctx context.Context, payload dtos.BulkUpdateMatchesResultDto) (*domain.SyncGroupStageOutcomes, error) {
	matchIDs, outcomes, err := s.prepareBulkUpdate(ctx, payload)
	if err != nil {
		return nil, err
	}

	s.asyncScoreMatches(outcomes, matchIDs)
	return outcomes, nil
}

// UpdateMatchResultsBulkSync mirrors UpdateMatchResultsBulk but blocks until
// the scoring chain (pickem + competition recompute, plus best-thirds when
// applicable) finishes. Intended for offline tools (seeders, replay scripts)
// that need deterministic ordering across multiple stage transitions in a row.
func (s *MatchService) UpdateMatchResultsBulkSync(ctx context.Context, payload dtos.BulkUpdateMatchesResultDto) (*domain.SyncGroupStageOutcomes, error) {
	matchIDs, outcomes, err := s.prepareBulkUpdate(ctx, payload)
	if err != nil {
		return nil, err
	}

	if err := s.scoreMatchOutcomes(ctx, outcomes, matchIDs); err != nil {
		return nil, err
	}

	return outcomes, nil
}

func (s *MatchService) prepareBulkUpdate(ctx context.Context, payload dtos.BulkUpdateMatchesResultDto) ([]int64, *domain.SyncGroupStageOutcomes, error) {
	matchIDs := make([]int64, 0, len(payload.Matches))
	for _, match := range payload.Matches {
		matchIDs = append(matchIDs, match.ID)
	}

	existingMatches, err := s.matchRepository.GetMatches(ctx, domain.MatchFilters{MatchIDs: matchIDs})
	if err != nil {
		return nil, nil, err
	}

	matchByID := make(map[int64]*domain.Match, len(existingMatches))
	for _, match := range existingMatches {
		matchByID[match.ID] = match
	}

	updates := make([]domain.MatchResultUpdate, 0, len(payload.Matches))
	for _, match := range payload.Matches {
		existing, ok := matchByID[match.ID]
		if !ok {
			return nil, nil, domain.ErrMatchesNotFound([]int64{match.ID})
		}
		if existing.Teams.Home == nil || existing.Teams.Away == nil {
			return nil, nil, domain.ErrMatchTeamsNotAssigned
		}
		if err := validateMatchResultForStage(existing.StageCode, match.UpdateMatchResultDto); err != nil {
			return nil, nil, err
		}
		updates = append(updates, domain.MatchResultUpdate{
			MatchID:          match.ID,
			HomeScore:        *match.HomeScore,
			AwayScore:        *match.AwayScore,
			HomePenaltyScore: match.HomePenaltyScore,
			AwayPenaltyScore: match.AwayPenaltyScore,
			Status:           domain.MatchStatusFinished,
		})
	}

	if err := s.matchRepository.UpdateMatchesResult(ctx, updates); err != nil {
		return nil, nil, err
	}

	outcomes, err := s.SyncGroupStageOutcomes(ctx)
	if err != nil {
		return nil, nil, err
	}

	return matchIDs, outcomes, nil
}

func (s *MatchService) ResetMatchResult(ctx context.Context, matchID int64) (*domain.SyncGroupStageOutcomes, error) {
	if err := s.matchRepository.ResetMatchResult(ctx, matchID); err != nil {
		return nil, err
	}

	return s.SyncGroupStageOutcomes(ctx)
	// Note: we deliberately do NOT trigger pickem scoring on reset
	// Resetting a result doesn't add new score events; an admin must explicitly
	// rescore after correcting and re-finalizing the match result
}

// scoreMatchOutcomes runs the post-result scoring chain synchronously: pickem
// scoring, competition match-score recompute, and best-thirds when group stage
// just completed. The HTTP request path wraps it in a goroutine via
// asyncScoreMatches; CLI tools that need deterministic ordering call it directly
func (s *MatchService) scoreMatchOutcomes(ctx context.Context, outcomes *domain.SyncGroupStageOutcomes, matchIDs []int64) error {
	result, err := s.scoringService.ScoreMatches(ctx, matchIDs)
	if err != nil {
		s.logger.Error("pickem scoring failed",
			logging.Error, err.Error(),
			"match_ids", matchIDs,
		)
		return err
	}

	if err := s.competitionScoringService.RecomputeForMatches(ctx, result); err != nil {
		s.logger.Error("competition scoring recompute failed",
			logging.Error, err.Error(),
			"match_ids", matchIDs,
		)
		return err
	}

	if outcomes != nil && outcomes.PromotionOutcome != nil &&
		outcomes.PromotionOutcome.Status == domain.PromotionStatusCompleted {
		bestThirdUserIDs, err := s.scoringService.ScoreBestThirds(ctx)
		if err != nil {
			s.logger.Error("pickem best-thirds scoring failed",
				logging.Error, err.Error(),
			)
			return err
		}

		if err := s.competitionScoringService.RecomputeForBestThirds(ctx, bestThirdUserIDs); err != nil {
			s.logger.Error("competition scoring recompute for best-thirds failed",
				logging.Error, err.Error(),
			)
			return err
		}
	}

	return nil
}

func (s *MatchService) asyncScoreMatches(outcomes *domain.SyncGroupStageOutcomes, matchIDs []int64) {
	go func() {
		_ = s.scoreMatchOutcomes(context.Background(), outcomes, matchIDs)
	}()
}

func (s *MatchService) SyncGroupStageOutcomes(ctx context.Context) (*domain.SyncGroupStageOutcomes, error) {
	if err := s.groupStandingService.RecalculateStandings(ctx); err != nil {
		return nil, err
	}

	if err := s.promoteGroupWinners(ctx); err != nil {
		return nil, err
	}

	isGroupStageFinished, err := s.matchRepository.IsGroupStageFinished(ctx)
	if err != nil {
		return nil, err
	}

	if !isGroupStageFinished {
		return &domain.SyncGroupStageOutcomes{
			IsGroupStageFinished: false,
		}, nil
	}

	promotionOutcome, err := s.promoteThirdPlaceTeams(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.SyncGroupStageOutcomes{
		IsGroupStageFinished: true,
		PromotionOutcome:     promotionOutcome,
	}, nil
}

func (s *MatchService) ResolveThirdPlaceConflict(ctx context.Context, payload dtos.ResolveThirdPlaceConflictDto) (*domain.SyncGroupStageOutcomes, error) {
	// Normalize and validate the team FIFA codes
	normalizedPayload := make([]string, 0, len(payload.TeamFifaCodes))
	seen := make(map[string]bool)

	for _, code := range payload.TeamFifaCodes {
		upperCode := strings.ToUpper(strings.TrimSpace(code))
		if upperCode == "" {
			return nil, domain.ErrThirdPlaceInvalidSelection
		}

		if seen[upperCode] {
			return nil, domain.ErrThirdPlaceInvalidSelection
		}

		seen[upperCode] = true
		normalizedPayload = append(normalizedPayload, upperCode)
	}

	if len(normalizedPayload) != 8 {
		return nil, domain.ErrThirdPlaceInvalidSelection
	}

	thirdPlaceTeams, err := s.groupStandingRepository.GetThirdPlaceGroups(ctx)
	if err != nil {
		return nil, err
	}

	inConflict, candidates := evaluateThirdPlaceCutoffConflict(thirdPlaceTeams)
	if !inConflict {
		return nil, domain.ErrThirdPlaceNotInConflict
	}

	// FIFA codes still allowed for this resolution (from the conflict candidate list).
	candidateFifa := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		candidateFifa[c.FifaCode] = true
	}

	// Map to look up third-place teams by FIFA code.
	thirdPlaceTeamsByFifaCode := make(map[string]*domain.ThirdPlaceTeam)
	for _, t := range thirdPlaceTeams {
		thirdPlaceTeamsByFifaCode[t.FifaCode] = t
	}

	// Check that the provided FIFA codes belong to the conflict candidate list.
	selected := make([]*domain.ThirdPlaceTeam, 0, 8)
	for _, code := range normalizedPayload {
		if !candidateFifa[code] {
			return nil, domain.ErrThirdPlaceInvalidSelection
		}

		selected = append(selected, thirdPlaceTeamsByFifaCode[code])
	}

	if _, err := s.applyThirdPlaceAssignments(ctx, selected); err != nil {
		return nil, err
	}

	outcomes, err := s.SyncGroupStageOutcomes(ctx)
	if err != nil {
		return nil, err
	}

	// Third-place qualifiers are now resolved -> best_third_pick events can be scored
	go func() {
		bestThirdUserIDs, err := s.scoringService.ScoreBestThirds(context.Background())
		if err != nil {
			s.logger.Error("pickem best-thirds scoring failed (post conflict resolve)",
				logging.Error, err.Error(),
			)
			return
		}

		if err := s.competitionScoringService.RecomputeForBestThirds(context.Background(), bestThirdUserIDs); err != nil {
			s.logger.Error("competition scoring recompute for best-thirds failed (post conflict resolve)",
				logging.Error, err.Error(),
			)
		}
	}()

	return outcomes, nil
}

func (s *MatchService) promoteGroupWinners(ctx context.Context) error {
	groupCodes := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	var allUpdates []domain.MatchTeamUpdate

	for _, groupCode := range groupCodes {
		wg.Add(1)
		go func(groupCode string) {
			defer wg.Done()

			isGroupFinished, err := s.matchRepository.IsGroupFinished(ctx, groupCode)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			if !isGroupFinished {
				return
			}

			standings, err := s.groupStandingRepository.GetGroupStandings(ctx, []string{groupCode}, nil)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			updates := buildGroupPositionMatchUpdates(groupCode, standings)
			mu.Lock()
			allUpdates = append(allUpdates, updates...)
			mu.Unlock()
		}(groupCode)
	}
	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("failed to promote group winners: %v", errs)
	}

	if len(allUpdates) == 0 {
		return nil
	}

	// Sort by matchID to prevent deadlocks
	sort.Slice(allUpdates, func(i, j int) bool {
		return allUpdates[i].MatchID < allUpdates[j].MatchID
	})

	return s.matchRepository.UpdateMatchTeams(ctx, allUpdates)
}

func (s *MatchService) promoteThirdPlaceTeams(ctx context.Context) (*domain.PromoteThirdPlaceTeams, error) {
	teams, err := s.groupStandingRepository.GetThirdPlaceGroups(ctx)
	if err != nil {
		return nil, err
	}

	inConflict, candidates := evaluateThirdPlaceCutoffConflict(teams)
	if inConflict {
		return &domain.PromoteThirdPlaceTeams{
			Status:     domain.PromotionStatusConflict,
			Candidates: candidates,
		}, nil
	}

	assignments, err := s.applyThirdPlaceAssignments(ctx, teams[:8])
	if err != nil {
		return nil, err
	}

	return &domain.PromoteThirdPlaceTeams{
		Status:      domain.PromotionStatusCompleted,
		Assignments: assignments,
	}, nil
}

func (s *MatchService) applyThirdPlaceAssignments(ctx context.Context, teams []*domain.ThirdPlaceTeam) ([]domain.ThirdPlaceAssignment, error) {
	qualifyingGroups := make([]string, len(teams))
	for i, team := range teams {
		qualifyingGroups[i] = team.GroupCode
	}

	sort.Strings(qualifyingGroups)
	combination := s.findCombination(qualifyingGroups)
	matchTeamUpdates := buildThirdPlaceMatchUpdates(combination.Assignments, teams)

	// Sort by matchID to prevent deadlocks when updating matches
	sort.Slice(matchTeamUpdates, func(i, j int) bool {
		return matchTeamUpdates[i].MatchID < matchTeamUpdates[j].MatchID
	})

	if err := s.matchRepository.UpdateMatchTeams(ctx, matchTeamUpdates); err != nil {
		return nil, err
	}

	assignments := make([]domain.ThirdPlaceAssignment, len(matchTeamUpdates))
	for i, update := range matchTeamUpdates {
		assignments[i] = domain.ThirdPlaceAssignment{
			MatchID:          update.MatchID,
			AwayTeamFifaCode: *update.AwayTeamFifaCode,
		}
	}

	return assignments, nil
}

func evaluateThirdPlaceCutoffConflict(teams []*domain.ThirdPlaceTeam) (bool, []domain.ThirdPlaceCandidate) {
	cutoffStart, cutoffEnd := thirdPlaceCutoffBounds(teams)
	cutoffLen := cutoffEnd - cutoffStart + 1
	// Eight third-place teams advance; the first `cutoffStart` list positions are already spoken for.
	// Everyone from the tie downward only has `8 - cutoffStart` slots left among those eight.
	availableSpots := 8 - cutoffStart

	if cutoffLen > availableSpots {
		return true, thirdPlaceCandidates(teams, cutoffStart, cutoffLen)
	}

	return false, nil
}

func (s *MatchService) findCombination(qualifyingGroups []string) *domain.ThirdPlaceCombination {
	for _, combo := range s.combinations {
		if slices.Equal(combo.QualifyingGroups, qualifyingGroups) {
			return &combo
		}
	}

	return nil
}

func thirdPlaceCutoffBounds(teams []*domain.ThirdPlaceTeam) (start, end int) {
	ref := teams[7]

	// Check upwards from the reference team
	start = 7
	for start > 0 {
		t := teams[start-1]
		if t.Points != ref.Points || t.GoalDifference != ref.GoalDifference || t.GoalsFor != ref.GoalsFor {
			break
		}
		start--
	}

	// Check downwards from the reference team
	end = 7
	for end < len(teams)-1 {
		t := teams[end+1]
		if t.Points != ref.Points || t.GoalDifference != ref.GoalDifference || t.GoalsFor != ref.GoalsFor {
			break
		}
		end++
	}

	// Returns inclusive [start, end] indices of every team tied with the
	// eighth-ranked third (index 7) on points, goal difference, and goals for the implemented tiebreakers.
	return start, end
}

// thirdPlaceCandidates returns every team that could still qualify with their
// current standing position: guaranteed qualifiers above the cutoff group plus
// the tied cutoff group itself. Teams ranked below the cutoff are already eliminated.
func thirdPlaceCandidates(teams []*domain.ThirdPlaceTeam, cutoffStartIndex int, cutoffGroupSize int) []domain.ThirdPlaceCandidate {
	candidates := teams[:cutoffStartIndex+cutoffGroupSize]
	result := make([]domain.ThirdPlaceCandidate, len(candidates))
	for i, t := range candidates {
		result[i] = domain.ThirdPlaceCandidate{
			Position: i + 1,
			FifaCode: t.FifaCode,
			IsTied:   i >= cutoffStartIndex,
		}
	}

	return result
}

func buildThirdPlaceMatchUpdates(assignments map[string]string, thirdPlaceTeams []*domain.ThirdPlaceTeam) []domain.MatchTeamUpdate {
	// Create map of group code to FIFA code for top 8 teams
	groupToFifaCode := make(map[string]string)
	for _, team := range thirdPlaceTeams {
		groupToFifaCode[team.GroupCode] = team.FifaCode
	}

	var matchTeamUpdates []domain.MatchTeamUpdate

	for firstPlace, thirdPlace := range assignments {
		firstPlaceGroupCode := string(firstPlace[1])
		thirdPlaceGroupCode := string(thirdPlace[1])
		thirdPlaceFifaCode := groupToFifaCode[thirdPlaceGroupCode]

		// Find the match in MatchSlotRules where this first-place team plays
		var matchID int64

		for id, rule := range domain.MatchSlotRules {
			// First place team is always going to be playing at Home
			if rule.Home.Kind == domain.SourceGroupPosition &&
				rule.Home.Position == 1 &&
				rule.Home.GroupCode == firstPlaceGroupCode {
				matchID = id
				break
			}
		}

		matchTeamUpdates = append(matchTeamUpdates, domain.MatchTeamUpdate{
			MatchID:          matchID,
			AwayTeamFifaCode: &thirdPlaceFifaCode,
		})
	}

	return matchTeamUpdates
}

func buildGroupPositionMatchUpdates(groupCode string, groupStandings []*domain.GroupStanding) []domain.MatchTeamUpdate {
	positionToTeamCode := make(map[int]string)
	for _, standing := range groupStandings {
		positionToTeamCode[standing.Position] = standing.Team.FifaCode
	}

	var matchTeamUpdates []domain.MatchTeamUpdate

	// Find matches referencing this group and source kind
	for matchID, rule := range domain.MatchSlotRules {
		if rule.Home.Kind == domain.SourceGroupPosition && rule.Home.GroupCode == groupCode {
			teamCode := positionToTeamCode[rule.Home.Position]
			matchTeamUpdates = append(matchTeamUpdates, domain.MatchTeamUpdate{
				MatchID:          matchID,
				HomeTeamFifaCode: &teamCode,
			})
		}

		if rule.Away.Kind == domain.SourceGroupPosition && rule.Away.GroupCode == groupCode {
			teamCode := positionToTeamCode[rule.Away.Position]
			matchTeamUpdates = append(matchTeamUpdates, domain.MatchTeamUpdate{
				MatchID:          matchID,
				AwayTeamFifaCode: &teamCode,
			})
		}
	}

	return matchTeamUpdates
}

func loadThirdPlaceCombinations() []domain.ThirdPlaceCombination {
	var combinations []domain.ThirdPlaceCombination
	if err := json.Unmarshal(combinationsJSON, &combinations); err != nil {
		panic("failed to parse embedded combinations.json: " + err.Error())
	}

	return combinations
}

func validateMatchResultForStage(stage domain.MatchStageCode, payload dtos.UpdateMatchResultDto) error {
	hasPenalty := payload.HomePenaltyScore != nil

	// If penalty input is provided on group-stage matches, reject it
	if !stage.IsKnockout() {
		if hasPenalty {
			return domain.ErrPenaltyForbidden
		}

		return nil
	}

	regularTied := *payload.HomeScore == *payload.AwayScore
	switch {
	// If the match is tied and no penalties are provided, reject it
	case regularTied && !hasPenalty:
		return domain.ErrPenaltyRequired
	// If the match is not tied and penalties are provided, reject it
	case !regularTied && hasPenalty:
		return domain.ErrPenaltyForbidden
	}

	return nil
}
