package services

import (
	"context"
	"strconv"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

var bestThirdSlotMatchIDs = []int64{74, 77, 79, 80, 81, 82, 85, 87}

type ScoringServiceInterface interface {
	ScoreMatches(ctx context.Context, matchIDs []int64) error
	ScoreBestThirds(ctx context.Context) error
}

type ScoringService struct {
	pickemRepository         domain.PickemRepository
	matchScorePickRepository domain.MatchScorePickRepository
	scoreEventRepository     domain.ScoreEventRepository
	userScoreRepository      domain.UserScoreRepository
	matchRepository          domain.MatchRepository
	groupStandingRepository  domain.GroupStandingRepository
	cfg                      *config.Config
	logger                   logging.Logger
}

func NewScoringService(
	pickemRepository domain.PickemRepository,
	matchScorePickRepository domain.MatchScorePickRepository,
	scoreEventRepository domain.ScoreEventRepository,
	userScoreRepository domain.UserScoreRepository,
	matchRepository domain.MatchRepository,
	groupStandingRepository domain.GroupStandingRepository,
	cfg *config.Config,
	logger logging.Logger,
) ScoringServiceInterface {
	return &ScoringService{
		pickemRepository:         pickemRepository,
		matchScorePickRepository: matchScorePickRepository,
		scoreEventRepository:     scoreEventRepository,
		userScoreRepository:      userScoreRepository,
		matchRepository:          matchRepository,
		groupStandingRepository:  groupStandingRepository,
		cfg:                      cfg,
		logger:                   logger,
	}
}

func (s *ScoringService) ScoreMatches(ctx context.Context, matchIDs []int64) error {
	if len(matchIDs) == 0 {
		return nil
	}

	matches, err := s.matchRepository.GetMatches(ctx, domain.MatchFilters{
		MatchIDs: matchIDs,
		Status:   domain.MatchStatusFinished,
	})
	if err != nil {
		s.logger.Error("failed to get matches",
			logging.Error, err.Error(),
			"match_ids", matchIDs,
		)
		return err
	}

	var allScoreEvents []*domain.ScoreEvent
	// Set of user IDs that were affected by the scoring run
	affectedUserIDs := make(map[string]struct{})
	seenGroups := make(map[string]struct{})

	for _, match := range matches {
		// Match score picks (every finished match, group or knockout)
		matchScoreEvents, matchScoreUserIDs, err := s.scoreMatchScorePicks(ctx, match)
		if err != nil {
			s.logger.Error("failed to score match score picks",
				logging.Error, err.Error(),
				"match_id", match.ID,
			)
			return err
		}

		allScoreEvents = append(allScoreEvents, matchScoreEvents...)
		addUserIDsToSet(affectedUserIDs, matchScoreUserIDs)

		// Group stage picks
		if match.StageCode == domain.MatchStageCodeGroupStage {
			groupCode := *match.GroupCode
			// If we've already scored this group, skip it
			if _, seen := seenGroups[groupCode]; seen {
				continue
			}
			// Mark this group as seen
			seenGroups[groupCode] = struct{}{}

			isGroupFinished, err := s.matchRepository.IsGroupFinished(ctx, groupCode)
			if err != nil {
				s.logger.Error("failed to check if group is finished",
					logging.Error, err.Error(),
					"group_code", groupCode,
					"match_id", match.ID,
				)
				return err
			}

			if isGroupFinished {
				groupScoreEvents, groupScoreUserIDs, err := s.scoreGroupStandingPicks(ctx, groupCode)
				if err != nil {
					s.logger.Error("failed to score group standing picks",
						logging.Error, err.Error(),
						"group_code", groupCode,
						"match_id", match.ID,
					)
					return err
				}

				allScoreEvents = append(allScoreEvents, groupScoreEvents...)
				addUserIDsToSet(affectedUserIDs, groupScoreUserIDs)
			}

			continue
		}

		// Bracket picks (knockout matches)
		bracketEvents, bracketUserIDs, err := s.scoreBracketPicks(ctx, match)
		if err != nil {
			s.logger.Error("failed to score bracket picks",
				logging.Error, err.Error(),
				"match_id", match.ID,
			)
			return err
		}

		allScoreEvents = append(allScoreEvents, bracketEvents...)
		addUserIDsToSet(affectedUserIDs, bracketUserIDs)
	}

	// Single batched score_events upsert
	if err := s.scoreEventRepository.BatchUpsertScoreEvents(ctx, allScoreEvents); err != nil {
		s.logger.Error("failed to upsert score events",
			logging.Error, err.Error(),
			"score_events_count", len(allScoreEvents),
			"affected_user_count", len(affectedUserIDs),
		)
		return err
	}

	// Single batched user_scores update
	userIDs := userIDSetToSlice(affectedUserIDs)
	if err := s.userScoreRepository.BatchUpdateUserScores(ctx, userIDs, s.cfg.Scoring.MatchScoreExact); err != nil {
		s.logger.Error("failed to update user scores",
			logging.Error, err.Error(),
			"affected_user_count", len(affectedUserIDs),
		)
		return err
	}

	s.logger.Info(
		"scoring run completed for match score picks and group standing picks",
		"match_count", len(matchIDs),
		"affected_user_count", len(affectedUserIDs),
	)

	return nil
}

func (s *ScoringService) ScoreBestThirds(ctx context.Context) error {
	// Derive the 8 actual advancing thirds from the populated R32 best-third slots.
	r32Matches, err := s.matchRepository.GetMatches(ctx, domain.MatchFilters{MatchIDs: bestThirdSlotMatchIDs})
	if err != nil {
		s.logger.Error("failed to get best third slot matches",
			logging.Error, err.Error(),
			"match_ids", bestThirdSlotMatchIDs,
		)
		return err
	}

	// Actual third teams that made it to the round of 32
	actualThirdTeams := make([]string, 0, 8)

	for _, match := range r32Matches {
		if match.Teams.Away != nil {
			actualThirdTeams = append(actualThirdTeams, match.Teams.Away.FifaCode)
		}
	}

	if len(actualThirdTeams) != 8 {
		s.logger.Error("best third teams count does not match expected",
			logging.Error, domain.ErrBestThirdsNotScoreable.Error(),
			"actual_third_teams_count", len(actualThirdTeams),
		)
		return domain.ErrBestThirdsNotScoreable
	}

	// Get users who picked the actual third teams
	bestThirdPicks, err := s.pickemRepository.GetBestThirdPicksByTeams(ctx, actualThirdTeams)
	if err != nil {
		s.logger.Error("failed to get best third picks by teams",
			logging.Error, err.Error(),
			"actual_third_teams_count", len(actualThirdTeams),
		)
		return err
	}

	var bestThirdScoreEvents []*domain.ScoreEvent
	// Set of user IDs that were affected by the scoring run
	affectedUserIDs := make(map[string]struct{})

	for _, pick := range bestThirdPicks {
		// Add the score event for the best third pick, points awarded for the correct pick
		bestThirdScoreEvents = append(bestThirdScoreEvents, &domain.ScoreEvent{
			UserID:     pick.UserID,
			SourceType: domain.ScoreSourceBestThirdPick,
			SourceRef:  pick.TeamFifaCode,
			Points:     s.cfg.Scoring.BestThird,
		})

		affectedUserIDs[pick.UserID] = struct{}{}
	}

	// Single batched score_events upsert
	if err := s.scoreEventRepository.BatchUpsertScoreEvents(ctx, bestThirdScoreEvents); err != nil {
		s.logger.Error("failed to upsert best third score events",
			logging.Error, err.Error(),
			"best_third_score_events_count", len(bestThirdScoreEvents),
			"affected_user_count", len(affectedUserIDs),
		)
		return err
	}

	// Single batched user_scores update
	userIDs := userIDSetToSlice(affectedUserIDs)
	if err := s.userScoreRepository.BatchUpdateUserScores(ctx, userIDs, s.cfg.Scoring.MatchScoreExact); err != nil {
		s.logger.Error("failed to update user scores for best third picks",
			logging.Error, err.Error(),
			"affected_user_count", len(affectedUserIDs),
		)
		return err
	}

	s.logger.Info(
		"scoring run completed for best third picks",
		"affected_user_count", len(affectedUserIDs),
	)

	return nil
}

func (s *ScoringService) scoreMatchScorePicks(ctx context.Context, match *domain.Match) ([]*domain.ScoreEvent, map[string]struct{}, error) {
	matchScorePicks, err := s.matchScorePickRepository.GetMatchScorePicksByMatch(ctx, match.ID)
	if err != nil {
		return nil, nil, err
	}

	// Actual scores and outcome
	actualHome := match.Result.HomeScore
	actualAway := match.Result.AwayScore
	actualOutcome := matchOutcome(actualHome, actualAway)

	scoreEvents := make([]*domain.ScoreEvent, 0, len(matchScorePicks))
	affectedUserIDs := make(map[string]struct{}, len(matchScorePicks))

	for _, pick := range matchScorePicks {
		var points int
		// Exact score pick
		if pick.HomeScore == actualHome && pick.AwayScore == actualAway {
			points = s.cfg.Scoring.MatchScoreExact
			// Correct outcome pick
		} else if matchOutcome(pick.HomeScore, pick.AwayScore) == actualOutcome {
			points = s.cfg.Scoring.MatchScoreOutcome
		}

		// If points were awarded to the pick, add the score event
		if points > 0 {
			scoreEvents = append(scoreEvents, &domain.ScoreEvent{
				UserID:     pick.UserID,
				SourceType: domain.ScoreSourceMatchScorePick,
				SourceRef:  strconv.FormatInt(match.ID, 10),
				Points:     points,
			})

			affectedUserIDs[pick.UserID] = struct{}{}
		}
	}

	// Return the score events and a Set of affected user IDs
	return scoreEvents, affectedUserIDs, nil
}

func (s *ScoringService) scoreGroupStandingPicks(ctx context.Context, groupCode string) ([]*domain.ScoreEvent, map[string]struct{}, error) {
	standings, err := s.groupStandingRepository.GetGroupStandings(ctx, []string{groupCode}, nil)
	if err != nil {
		return nil, nil, err
	}

	// Actual positions and qualifiers
	actualPosition := make(map[string]int, 4)
	qualifiers := make(map[string]struct{}, 2)

	for _, standing := range standings {
		// Map team positions to their FIFA code
		actualPosition[standing.Team.FifaCode] = standing.Position

		// If the team is in the top 2, they qualified for the next round
		if standing.Position <= 2 {
			qualifiers[standing.Team.FifaCode] = struct{}{}
		}
	}

	groupPicks, err := s.pickemRepository.GetGroupPicksByGroup(ctx, groupCode)
	if err != nil {
		return nil, nil, err
	}

	groupEvents := make([]*domain.ScoreEvent, 0, len(groupPicks))
	affectedUserIDs := make(map[string]struct{}, len(groupPicks))

	for _, pick := range groupPicks {
		// If the team is not in the actual position, skip it, no points awarded
		actual, ok := actualPosition[pick.TeamFifaCode]
		if !ok {
			continue
		}

		// If the team is in the top 2, they qualified for the next round, points awarded
		_, isQualifier := qualifiers[pick.TeamFifaCode]

		var points int
		// Exact position pick
		if pick.PredictedPosition == actual {
			points = s.cfg.Scoring.GroupPositionExact
			// Wrong position pick but qualified for the next round, points awarded
		} else if pick.PredictedPosition <= 2 && isQualifier {
			points = s.cfg.Scoring.GroupQualifies
		}

		// If points were awarded to the pick, add the score event
		if points > 0 {
			groupEvents = append(groupEvents, &domain.ScoreEvent{
				UserID:     pick.UserID,
				SourceType: domain.ScoreSourceGroupStandingPick,
				SourceRef:  groupCode + ":" + pick.TeamFifaCode,
				Points:     points,
			})

			affectedUserIDs[pick.UserID] = struct{}{}
		}
	}

	return groupEvents, affectedUserIDs, nil
}

func (s *ScoringService) scoreBracketPicks(ctx context.Context, match *domain.Match) ([]*domain.ScoreEvent, map[string]struct{}, error) {
	bracketPicks, err := s.pickemRepository.GetBracketPicksByMatch(ctx, match.ID)
	if err != nil {
		return nil, nil, err
	}

	points := bracketPointsForStage(match.StageCode, &s.cfg.Scoring)
	if points <= 0 {
		return nil, nil, nil
	}

	bracketEvents := make([]*domain.ScoreEvent, 0, len(bracketPicks))
	affectedUserIDs := make(map[string]struct{}, len(bracketPicks))

	for _, pick := range bracketPicks {
		// If the picked team is the winner, points awarded
		if pick.TeamFifaCode == *match.Result.WinnerTeamFifaCode {
			bracketEvents = append(bracketEvents, &domain.ScoreEvent{
				UserID:     pick.UserID,
				SourceType: domain.ScoreSourceBracketPick,
				SourceRef:  strconv.FormatInt(match.ID, 10),
				Points:     points,
			})

			affectedUserIDs[pick.UserID] = struct{}{}
		}
	}

	return bracketEvents, affectedUserIDs, nil
}

type matchResultOutcome string

const (
	outcomeHome matchResultOutcome = "home"
	outcomeAway matchResultOutcome = "away"
	outcomeDraw matchResultOutcome = "draw"
)

func matchOutcome(home, away int) matchResultOutcome {
	if home > away {
		return outcomeHome
	}

	if away > home {
		return outcomeAway
	}

	return outcomeDraw
}

func bracketPointsForStage(stage domain.MatchStageCode, cfg *config.ScoringConfig) int {
	switch stage {
	case domain.MatchStageCodeRoundOf32:
		return cfg.RoundOf32
	case domain.MatchStageCodeRoundOf16:
		return cfg.RoundOf16
	case domain.MatchStageCodeQuarterFinals:
		return cfg.Quarterfinals
	case domain.MatchStageCodeSemiFinals:
		return cfg.Semifinals
	case domain.MatchStageCodeThirdPlace:
		return cfg.ThirdPlace
	case domain.MatchStageCodeFinal:
		return cfg.Final
	}
	return 0
}

func addUserIDsToSet(target, source map[string]struct{}) {
	for userID := range source {
		target[userID] = struct{}{}
	}
}

func userIDSetToSlice(userIDSet map[string]struct{}) []string {
	userIDs := make([]string, 0, len(userIDSet))

	for userID := range userIDSet {
		userIDs = append(userIDs, userID)
	}

	return userIDs
}
