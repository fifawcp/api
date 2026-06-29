package services

import (
	"context"
	"errors"
	"testing"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

func newTestMatchService(
	matchRepository *mocks.MockMatchRepository,
	groupStandingRepository *mocks.MockGroupStandingRepository,
	groupStandingService *mocks.MockGroupStandingService,
	scoringService *mocks.MockScoringService,
	competitionScoringService *mocks.MockCompetitionScoringService,
	logger *mocks.MockLogger,
) MatchServiceInterface {
	return NewMatchService(
		matchRepository,
		groupStandingRepository,
		groupStandingService,
		scoringService,
		competitionScoringService,
		logger,
	)
}

// thirdPlaceTeamsNoConflict returns 8 third-place teams with clearly distinct rankings
// (groups A-C-D-E-H-I-J-B, matching combination 470 for tests that exercise third-place promotion)
func thirdPlaceTeamsNoConflict() []*domain.ThirdPlaceTeam {
	return []*domain.ThirdPlaceTeam{
		{FifaCode: "BRA", GroupCode: "C", Points: 9, GoalDifference: 6, GoalsFor: 10},
		{FifaCode: "GER", GroupCode: "E", Points: 7, GoalDifference: 3, GoalsFor: 8},
		{FifaCode: "FRA", GroupCode: "I", Points: 6, GoalDifference: 2, GoalsFor: 7},
		{FifaCode: "ESP", GroupCode: "H", Points: 5, GoalDifference: 1, GoalsFor: 6},
		{FifaCode: "ARG", GroupCode: "J", Points: 4, GoalDifference: 0, GoalsFor: 5},
		{FifaCode: "MEX", GroupCode: "A", Points: 3, GoalDifference: 1, GoalsFor: 4},
		{FifaCode: "USA", GroupCode: "D", Points: 3, GoalDifference: 1, GoalsFor: 4},
		{FifaCode: "CAN", GroupCode: "B", Points: 2, GoalDifference: -2, GoalsFor: 3},
	}
}

// thirdPlaceTeamsWithConflict returns 12 third-place teams where teams at positions 7 and 8
// (POR from group I and NED from group A) are tied on all tiebreakers, creating a conflict.
// The 7 guaranteed qualifiers come from groups E,F,G,H,J,K,L; the tied pair is I and A.
// Selecting POR (group I) yields qualifying groups E,F,G,H,I,J,K,L → combination 1.
func thirdPlaceTeamsWithConflict() []*domain.ThirdPlaceTeam {
	return []*domain.ThirdPlaceTeam{
		{FifaCode: "BRA", GroupCode: "E", Points: 9, GoalDifference: 6, GoalsFor: 10},
		{FifaCode: "GER", GroupCode: "F", Points: 8, GoalDifference: 5, GoalsFor: 9},
		{FifaCode: "FRA", GroupCode: "G", Points: 7, GoalDifference: 4, GoalsFor: 8},
		{FifaCode: "ESP", GroupCode: "H", Points: 6, GoalDifference: 3, GoalsFor: 7},
		{FifaCode: "ARG", GroupCode: "J", Points: 5, GoalDifference: 2, GoalsFor: 6},
		{FifaCode: "MEX", GroupCode: "K", Points: 4, GoalDifference: 1, GoalsFor: 5},
		{FifaCode: "ENG", GroupCode: "L", Points: 3, GoalDifference: 1, GoalsFor: 4},
		{FifaCode: "POR", GroupCode: "I", Points: 2, GoalDifference: 0, GoalsFor: 3},
		{FifaCode: "NED", GroupCode: "A", Points: 2, GoalDifference: 0, GoalsFor: 3},
		{FifaCode: "ITA", GroupCode: "B", Points: 1, GoalDifference: -1, GoalsFor: 2},
		{FifaCode: "CHI", GroupCode: "C", Points: 1, GoalDifference: -2, GoalsFor: 1},
		{FifaCode: "URU", GroupCode: "D", Points: 0, GoalDifference: -3, GoalsFor: 0},
	}
}

func groupStageMatch() *domain.Match {
	return &domain.Match{
		ID: 1,
		Teams: domain.MatchTeams{
			Home: &domain.Team{FifaCode: "MEX"},
			Away: &domain.Team{FifaCode: "USA"},
		},
		StageCode: domain.MatchStageCodeGroupStage,
		Status:    domain.MatchStatusFinished,
		Result:    &domain.MatchResult{HomeScore: 1, AwayScore: 0},
	}
}

// knockoutMatch73CanBeatRsa is the finished Round-of-32 match 73 (RSA 0 - CAN 1).
// Its winner (CAN) feeds the home slot of Round-of-16 match 90 via MatchSlotRules.
func knockoutMatch73CanBeatRsa() *domain.Match {
	winner := "CAN"
	return &domain.Match{
		ID:        73,
		StageCode: domain.MatchStageCodeRoundOf32,
		Status:    domain.MatchStatusFinished,
		Teams: domain.MatchTeams{
			Home: &domain.Team{FifaCode: "RSA"},
			Away: &domain.Team{FifaCode: "CAN"},
		},
		Result: &domain.MatchResult{HomeScore: 0, AwayScore: 1, WinnerTeamFifaCode: &winner},
	}
}

// ---------------------------------------------------------------------------
// TestMatchService_GetMatches
// ---------------------------------------------------------------------------
func TestMatchService_GetMatches(t *testing.T) {
	t.Parallel()

	t.Run("returns matches", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{
					{ID: 1, Teams: domain.MatchTeams{Home: &domain.Team{FifaCode: "MEX"}, Away: &domain.Team{FifaCode: "USA"}}},
				}, nil
			},
		}

		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		matches, err := service.GetMatches(context.Background(), domain.MatchFilters{MatchIDs: []int64{1}})

		assert.NoError(t, err)
		assert.Equal(t, matches, []*domain.Match{
			{ID: 1, Teams: domain.MatchTeams{Home: &domain.Team{FifaCode: "MEX"}, Away: &domain.Team{FifaCode: "USA"}}},
		})
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return nil, errors.New("database error")
			},
		}

		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		matches, err := service.GetMatches(context.Background(), domain.MatchFilters{MatchIDs: []int64{1}})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, matches)
	})
}

// ---------------------------------------------------------------------------
// TestValidateMatchResultForStage
// ---------------------------------------------------------------------------
func TestValidateMatchResultForStage(t *testing.T) {
	t.Parallel()

	t.Run("group stage without penalty returns nil", func(t *testing.T) {
		t.Parallel()
		home, away := 1, 0
		err := validateMatchResultForStage(domain.MatchStageCodeGroupStage, dtos.UpdateMatchResultDto{
			HomeScore: &home,
			AwayScore: &away,
		})
		assert.NoError(t, err)
	})

	t.Run("group stage with penalty returns ErrPenaltyForbidden", func(t *testing.T) {
		t.Parallel()
		home, away, homePenalty := 1, 0, 5
		err := validateMatchResultForStage(domain.MatchStageCodeGroupStage, dtos.UpdateMatchResultDto{
			HomeScore:        &home,
			AwayScore:        &away,
			HomePenaltyScore: &homePenalty,
		})
		assert.ErrorIs(t, err, domain.ErrPenaltyForbidden)
	})

	t.Run("knockout tied with penalty returns nil", func(t *testing.T) {
		t.Parallel()
		home, away, homePenalty, awayPenalty := 1, 1, 5, 4
		err := validateMatchResultForStage(domain.MatchStageCodeRoundOf32, dtos.UpdateMatchResultDto{
			HomeScore:        &home,
			AwayScore:        &away,
			HomePenaltyScore: &homePenalty,
			AwayPenaltyScore: &awayPenalty,
		})
		assert.NoError(t, err)
	})

	t.Run("knockout tied without penalty returns ErrPenaltyRequired", func(t *testing.T) {
		t.Parallel()
		home, away := 1, 1
		err := validateMatchResultForStage(domain.MatchStageCodeRoundOf32, dtos.UpdateMatchResultDto{
			HomeScore: &home,
			AwayScore: &away,
		})
		assert.ErrorIs(t, err, domain.ErrPenaltyRequired)
	})

	t.Run("knockout decided with penalty returns ErrPenaltyForbidden", func(t *testing.T) {
		t.Parallel()
		home, away, homePenalty := 2, 1, 5
		err := validateMatchResultForStage(domain.MatchStageCodeRoundOf32, dtos.UpdateMatchResultDto{
			HomeScore:        &home,
			AwayScore:        &away,
			HomePenaltyScore: &homePenalty,
		})
		assert.ErrorIs(t, err, domain.ErrPenaltyForbidden)
	})

	t.Run("knockout decided without penalty returns nil", func(t *testing.T) {
		t.Parallel()
		home, away := 2, 1
		err := validateMatchResultForStage(domain.MatchStageCodeRoundOf32, dtos.UpdateMatchResultDto{
			HomeScore: &home,
			AwayScore: &away,
		})
		assert.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// TestMatchService_UpdateMatchResult
// ---------------------------------------------------------------------------
func TestMatchService_UpdateMatchResult(t *testing.T) {
	t.Parallel()

	t.Run("returns IsGroupStageFinished set to true and PromotionOutcome Status set to completed and fires pickem scoring on success when group is finished", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{groupStageMatch()}, nil
			},
			UpdateMatchesResultFunc: func(ctx context.Context, updates []domain.MatchResultUpdate) error {
				return nil
			},
			IsGroupStageFinishedFunc: func(ctx context.Context) (bool, error) {
				return true, nil
			},
			IsGroupFinishedFunc: func(ctx context.Context, groupCode string) (bool, error) {
				return true, nil
			},
			UpdateMatchTeamsFunc: func(ctx context.Context, updates []domain.MatchTeamUpdate) error {
				return nil
			},
		}

		gss := &mocks.MockGroupStandingService{
			RecalculateStandingsFunc: func(ctx context.Context) error {
				return nil
			},
		}

		ss := &mocks.MockScoringService{
			ScoreMatchesFunc: func(ctx context.Context, matchIDs []int64) (*domain.ScoreMatchesResult, error) {
				return &domain.ScoreMatchesResult{}, nil
			},
			ScoreBestThirdsFunc: func(ctx context.Context) ([]string, error) {
				return nil, nil
			},
		}

		gsr := &mocks.MockGroupStandingRepository{
			GetGroupStandingsFunc: func(ctx context.Context, groupCodes []string, position *int64) ([]*domain.GroupStanding, error) {
				return []*domain.GroupStanding{
					{
						Position:       1,
						Team:           domain.Team{FifaCode: "MEX"},
						MatchesPlayed:  1,
						Wins:           1,
						GoalsFor:       1,
						GoalDifference: 1,
						Points:         3,
					},
				}, nil
			},
			GetThirdPlaceGroupsFunc: func(ctx context.Context) ([]*domain.ThirdPlaceTeam, error) {
				return thirdPlaceTeamsNoConflict(), nil
			},
		}

		service := newTestMatchService(mr, gsr, gss, ss, &mocks.MockCompetitionScoringService{
			RecomputeForMatchesFunc:    func(ctx context.Context, result *domain.ScoreMatchesResult) error { return nil },
		}, nil)

		home, away := 1, 0
		syncGroupStageOutcomes, err := service.UpdateMatchResult(context.Background(), 1, dtos.UpdateMatchResultDto{
			HomeScore: &home,
			AwayScore: &away,
		})

		assert.NoError(t, err)
		assert.Equal(t, syncGroupStageOutcomes.IsGroupStageFinished, true)
		assert.Equal(t, syncGroupStageOutcomes.PromotionOutcome.Status, domain.PromotionStatusCompleted)
		assert.Equal(t, syncGroupStageOutcomes.PromotionOutcome.Assignments, []domain.ThirdPlaceAssignment{
			{MatchID: 74, AwayTeamFifaCode: "BRA"},
			{MatchID: 77, AwayTeamFifaCode: "USA"},
			{MatchID: 79, AwayTeamFifaCode: "ESP"},
			{MatchID: 80, AwayTeamFifaCode: "FRA"},
			{MatchID: 81, AwayTeamFifaCode: "CAN"},
			{MatchID: 82, AwayTeamFifaCode: "MEX"},
			{MatchID: 85, AwayTeamFifaCode: "ARG"},
			{MatchID: 87, AwayTeamFifaCode: "GER"},
		})
	})

	t.Run("returns IsGroupStageFinished set to false and fires pickem scoring on success when input group is not finished", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{groupStageMatch()}, nil
			},
			UpdateMatchesResultFunc: func(ctx context.Context, updates []domain.MatchResultUpdate) error {
				return nil
			},
			IsGroupStageFinishedFunc: func(ctx context.Context) (bool, error) {
				return false, nil
			},
			IsGroupFinishedFunc: func(ctx context.Context, groupCode string) (bool, error) {
				return false, nil
			},
		}

		gss := &mocks.MockGroupStandingService{
			RecalculateStandingsFunc: func(ctx context.Context) error {
				return nil
			},
		}

		ss := &mocks.MockScoringService{
			ScoreMatchesFunc: func(ctx context.Context, matchIDs []int64) (*domain.ScoreMatchesResult, error) {
				return &domain.ScoreMatchesResult{}, nil
			},
		}

		service := newTestMatchService(mr, nil, gss, ss, &mocks.MockCompetitionScoringService{
			RecomputeForMatchesFunc:    func(ctx context.Context, result *domain.ScoreMatchesResult) error { return nil },
		}, nil)

		home, away := 1, 0
		syncGroupStageOutcomes, err := service.UpdateMatchResult(context.Background(), 1, dtos.UpdateMatchResultDto{
			HomeScore: &home,
			AwayScore: &away,
		})

		assert.NoError(t, err)
		assert.Equal(t, syncGroupStageOutcomes, &domain.SyncGroupStageOutcomes{
			IsGroupStageFinished: false,
		})
	})

	t.Run("returns ErrMatchNotFound when match does not exist", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{}, nil
			},
		}

		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		home, away := 1, 0
		outcomes, err := service.UpdateMatchResult(context.Background(), 99, dtos.UpdateMatchResultDto{
			HomeScore: &home,
			AwayScore: &away,
		})

		assert.ErrorIs(t, err, domain.ErrMatchNotFound)
		assert.Nil(t, outcomes)
	})

	t.Run("returns ErrPenaltyForbidden for group stage match with penalty", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{groupStageMatch()}, nil
			},
		}

		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		home, away, homePenalty := 1, 0, 5
		outcomes, err := service.UpdateMatchResult(context.Background(), 1, dtos.UpdateMatchResultDto{
			HomeScore:        &home,
			AwayScore:        &away,
			HomePenaltyScore: &homePenalty,
		})

		assert.ErrorIs(t, err, domain.ErrPenaltyForbidden)
		assert.Nil(t, outcomes)
	})

	t.Run("returns ErrMatchTeamsNotAssigned for knockout match without contenders", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{
					{
						ID:        79,
						StageCode: domain.MatchStageCodeRoundOf32,
						Status:    domain.MatchStatusScheduled,
						Teams:     domain.MatchTeams{Home: nil, Away: nil},
					},
				}, nil
			},
		}

		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		home, away, homePenalty, awayPenalty := 0, 0, 1, 2
		outcomes, err := service.UpdateMatchResult(context.Background(), 79, dtos.UpdateMatchResultDto{
			HomeScore:        &home,
			AwayScore:        &away,
			HomePenaltyScore: &homePenalty,
			AwayPenaltyScore: &awayPenalty,
		})

		assert.ErrorIs(t, err, domain.ErrMatchTeamsNotAssigned)
		assert.Nil(t, outcomes)
	})

	t.Run("propagates repository update error", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{groupStageMatch()}, nil
			},
			UpdateMatchesResultFunc: func(ctx context.Context, updates []domain.MatchResultUpdate) error {
				return errors.New("db write error")
			},
		}

		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		home, away := 1, 0
		outcomes, err := service.UpdateMatchResult(context.Background(), 1, dtos.UpdateMatchResultDto{
			HomeScore: &home,
			AwayScore: &away,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db write error")
		assert.Nil(t, outcomes)
	})

	t.Run("advances bracket and skips group sync for a knockout match", func(t *testing.T) {
		t.Parallel()

		var capturedTeams []domain.MatchTeamUpdate
		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{knockoutMatch73CanBeatRsa()}, nil
			},
			UpdateMatchesResultFunc: func(ctx context.Context, updates []domain.MatchResultUpdate) error {
				return nil
			},
			UpdateMatchTeamsFunc: func(ctx context.Context, updates []domain.MatchTeamUpdate) error {
				capturedTeams = updates
				return nil
			},
			// IsGroupStageFinishedFunc / IsGroupFinishedFunc / RecalculateStandingsFunc are
			// intentionally unset: reaching SyncGroupStageOutcomes would panic and fail this test.
		}

		ss := &mocks.MockScoringService{
			ScoreMatchesFunc: func(ctx context.Context, matchIDs []int64) (*domain.ScoreMatchesResult, error) {
				return &domain.ScoreMatchesResult{}, nil
			},
		}

		service := newTestMatchService(mr, nil, nil, ss, &mocks.MockCompetitionScoringService{
			RecomputeForMatchesFunc: func(ctx context.Context, result *domain.ScoreMatchesResult) error { return nil },
		}, nil)

		home, away := 0, 1
		outcomes, err := service.UpdateMatchResult(context.Background(), 73, dtos.UpdateMatchResultDto{
			HomeScore: &home,
			AwayScore: &away,
		})

		assert.NoError(t, err)
		assert.Equal(t, &domain.SyncGroupStageOutcomes{IsGroupStageFinished: true}, outcomes)
		assert.Len(t, capturedTeams, 1)
		assert.Equal(t, int64(90), capturedTeams[0].MatchID)
		assert.Equal(t, "CAN", *capturedTeams[0].HomeTeamFifaCode)
		assert.Nil(t, capturedTeams[0].AwayTeamFifaCode) // match 75 not finished yet
	})
}

// ---------------------------------------------------------------------------
// TestMatchService_UpdateMatchResultsBulk
// ---------------------------------------------------------------------------
func TestMatchService_UpdateMatchResultsBulk(t *testing.T) {
	t.Parallel()

	t.Run("returns IsGroupStageFinished set to false when group is not finished", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{groupStageMatch()}, nil
			},
			UpdateMatchesResultFunc: func(ctx context.Context, updates []domain.MatchResultUpdate) error {
				return nil
			},
			IsGroupFinishedFunc: func(ctx context.Context, groupCode string) (bool, error) {
				return false, nil
			},
			IsGroupStageFinishedFunc: func(ctx context.Context) (bool, error) {
				return false, nil
			},
		}

		gss := &mocks.MockGroupStandingService{
			RecalculateStandingsFunc: func(ctx context.Context) error { return nil },
		}

		ss := &mocks.MockScoringService{
			ScoreMatchesFunc: func(ctx context.Context, matchIDs []int64) (*domain.ScoreMatchesResult, error) {
				return &domain.ScoreMatchesResult{}, nil
			},
		}

		service := newTestMatchService(mr, nil, gss, ss, &mocks.MockCompetitionScoringService{
			RecomputeForMatchesFunc:    func(ctx context.Context, result *domain.ScoreMatchesResult) error { return nil },
		}, nil)

		home, away := 1, 0
		outcomes, err := service.UpdateMatchResultsBulk(context.Background(), dtos.BulkUpdateMatchesResultDto{
			Matches: []dtos.BulkUpdateMatchResultDto{
				{ID: 1, UpdateMatchResultDto: dtos.UpdateMatchResultDto{HomeScore: &home, AwayScore: &away}},
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, outcomes, &domain.SyncGroupStageOutcomes{IsGroupStageFinished: false})
	})

	t.Run("returns IsGroupStageFinished set to true with PromotionStatusCompleted when group stage finishes", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{groupStageMatch()}, nil
			},
			UpdateMatchesResultFunc: func(ctx context.Context, updates []domain.MatchResultUpdate) error {
				return nil
			},
			IsGroupFinishedFunc: func(ctx context.Context, groupCode string) (bool, error) {
				return false, nil
			},
			IsGroupStageFinishedFunc: func(ctx context.Context) (bool, error) {
				return true, nil
			},
			UpdateMatchTeamsFunc: func(ctx context.Context, updates []domain.MatchTeamUpdate) error {
				return nil
			},
		}

		gss := &mocks.MockGroupStandingService{
			RecalculateStandingsFunc: func(ctx context.Context) error { return nil },
		}

		ss := &mocks.MockScoringService{
			ScoreMatchesFunc: func(ctx context.Context, matchIDs []int64) (*domain.ScoreMatchesResult, error) {
				return &domain.ScoreMatchesResult{}, nil
			},
			ScoreBestThirdsFunc: func(ctx context.Context) ([]string, error) { return nil, nil },
		}

		gsr := &mocks.MockGroupStandingRepository{
			GetThirdPlaceGroupsFunc: func(ctx context.Context) ([]*domain.ThirdPlaceTeam, error) {
				return thirdPlaceTeamsNoConflict(), nil
			},
		}

		service := newTestMatchService(mr, gsr, gss, ss, &mocks.MockCompetitionScoringService{
			RecomputeForMatchesFunc:    func(ctx context.Context, result *domain.ScoreMatchesResult) error { return nil },
		}, nil)

		home, away := 1, 0
		outcomes, err := service.UpdateMatchResultsBulk(context.Background(), dtos.BulkUpdateMatchesResultDto{
			Matches: []dtos.BulkUpdateMatchResultDto{
				{ID: 1, UpdateMatchResultDto: dtos.UpdateMatchResultDto{HomeScore: &home, AwayScore: &away}},
			},
		})

		assert.NoError(t, err)
		assert.True(t, outcomes.IsGroupStageFinished)
		assert.Equal(t, outcomes.PromotionOutcome.Status, domain.PromotionStatusCompleted)
	})

	t.Run("advances bracket and skips group sync for a knockout-only payload", func(t *testing.T) {
		t.Parallel()

		var capturedTeams []domain.MatchTeamUpdate
		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{knockoutMatch73CanBeatRsa()}, nil
			},
			UpdateMatchesResultFunc: func(ctx context.Context, updates []domain.MatchResultUpdate) error {
				return nil
			},
			UpdateMatchTeamsFunc: func(ctx context.Context, updates []domain.MatchTeamUpdate) error {
				capturedTeams = updates
				return nil
			},
			// Group-sync mocks intentionally unset; SyncGroupStageOutcomes must not run.
		}

		ss := &mocks.MockScoringService{
			ScoreMatchesFunc: func(ctx context.Context, matchIDs []int64) (*domain.ScoreMatchesResult, error) {
				return &domain.ScoreMatchesResult{}, nil
			},
		}

		service := newTestMatchService(mr, nil, nil, ss, &mocks.MockCompetitionScoringService{
			RecomputeForMatchesFunc: func(ctx context.Context, result *domain.ScoreMatchesResult) error { return nil },
		}, nil)

		home, away := 0, 1
		outcomes, err := service.UpdateMatchResultsBulk(context.Background(), dtos.BulkUpdateMatchesResultDto{
			Matches: []dtos.BulkUpdateMatchResultDto{
				{ID: 73, UpdateMatchResultDto: dtos.UpdateMatchResultDto{HomeScore: &home, AwayScore: &away}},
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, &domain.SyncGroupStageOutcomes{IsGroupStageFinished: true}, outcomes)
		assert.Len(t, capturedTeams, 1)
		assert.Equal(t, int64(90), capturedTeams[0].MatchID)
		assert.Equal(t, "CAN", *capturedTeams[0].HomeTeamFifaCode)
	})

	t.Run("returns ErrMatchesNotFound when a match ID is not in the database", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{}, nil
			},
		}

		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		home, away := 1, 0
		outcomes, err := service.UpdateMatchResultsBulk(context.Background(), dtos.BulkUpdateMatchesResultDto{
			Matches: []dtos.BulkUpdateMatchResultDto{
				{ID: 99, UpdateMatchResultDto: dtos.UpdateMatchResultDto{HomeScore: &home, AwayScore: &away}},
			},
		})

		assert.Error(t, err)
		assert.ErrorAs(t, err, &domain.MatchesNotFoundError{})
		assert.Nil(t, outcomes)
	})

	t.Run("returns ErrPenaltyForbidden for group stage match with penalty", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{groupStageMatch()}, nil
			},
		}

		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		home, away, homePenalty := 1, 0, 5
		outcomes, err := service.UpdateMatchResultsBulk(context.Background(), dtos.BulkUpdateMatchesResultDto{
			Matches: []dtos.BulkUpdateMatchResultDto{
				{ID: 1, UpdateMatchResultDto: dtos.UpdateMatchResultDto{
					HomeScore:        &home,
					AwayScore:        &away,
					HomePenaltyScore: &homePenalty,
				}},
			},
		})

		assert.ErrorIs(t, err, domain.ErrPenaltyForbidden)
		assert.Nil(t, outcomes)
	})

	t.Run("returns ErrMatchTeamsNotAssigned for knockout match without contenders", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{
					{
						ID:        79,
						StageCode: domain.MatchStageCodeRoundOf32,
						Status:    domain.MatchStatusScheduled,
						Teams:     domain.MatchTeams{Home: nil, Away: nil},
					},
				}, nil
			},
		}

		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		home, away, homePenalty, awayPenalty := 0, 0, 1, 2
		outcomes, err := service.UpdateMatchResultsBulk(context.Background(), dtos.BulkUpdateMatchesResultDto{
			Matches: []dtos.BulkUpdateMatchResultDto{
				{ID: 79, UpdateMatchResultDto: dtos.UpdateMatchResultDto{
					HomeScore:        &home,
					AwayScore:        &away,
					HomePenaltyScore: &homePenalty,
					AwayPenaltyScore: &awayPenalty,
				}},
			},
		})

		assert.ErrorIs(t, err, domain.ErrMatchTeamsNotAssigned)
		assert.Nil(t, outcomes)
	})

	t.Run("propagates repository update error", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{groupStageMatch()}, nil
			},
			UpdateMatchesResultFunc: func(ctx context.Context, updates []domain.MatchResultUpdate) error {
				return errors.New("db write error")
			},
		}

		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		home, away := 1, 0
		outcomes, err := service.UpdateMatchResultsBulk(context.Background(), dtos.BulkUpdateMatchesResultDto{
			Matches: []dtos.BulkUpdateMatchResultDto{
				{ID: 1, UpdateMatchResultDto: dtos.UpdateMatchResultDto{HomeScore: &home, AwayScore: &away}},
			},
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db write error")
		assert.Nil(t, outcomes)
	})
}

// ---------------------------------------------------------------------------
// TestMatchService_ResetMatchResult
// ---------------------------------------------------------------------------
func TestMatchService_ResetMatchResult(t *testing.T) {
	t.Parallel()

	t.Run("resets result and returns outcomes without triggering scoring", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			ResetMatchResultFunc: func(ctx context.Context, matchID int64) error {
				return nil
			},
			IsGroupFinishedFunc: func(ctx context.Context, groupCode string) (bool, error) {
				return false, nil
			},
			IsGroupStageFinishedFunc: func(ctx context.Context) (bool, error) {
				return false, nil
			},
		}

		gss := &mocks.MockGroupStandingService{
			RecalculateStandingsFunc: func(ctx context.Context) error { return nil },
		}

		// No scoring mocks set — any call to ScoreMatches/ScoreBestThirds would panic,
		// confirming that reset does not trigger pick'em scoring.
		service := newTestMatchService(mr, nil, gss, nil, nil, nil)

		outcomes, err := service.ResetMatchResult(context.Background(), 1)

		assert.NoError(t, err)
		assert.Equal(t, outcomes, &domain.SyncGroupStageOutcomes{IsGroupStageFinished: false})
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			ResetMatchResultFunc: func(ctx context.Context, matchID int64) error {
				return errors.New("db reset error")
			},
		}

		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		outcomes, err := service.ResetMatchResult(context.Background(), 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db reset error")
		assert.Nil(t, outcomes)
	})
}

// ---------------------------------------------------------------------------
// TestMatchService_ResolveThirdPlaceConflict
// ---------------------------------------------------------------------------
func TestMatchService_ResolveThirdPlaceConflict(t *testing.T) {
	t.Parallel()

	t.Run("resolves conflict and returns group stage finished outcomes", func(t *testing.T) {
		t.Parallel()

		conflictTeams := thirdPlaceTeamsWithConflict()

		mr := &mocks.MockMatchRepository{
			UpdateMatchTeamsFunc: func(ctx context.Context, updates []domain.MatchTeamUpdate) error {
				return nil
			},
			IsGroupFinishedFunc: func(ctx context.Context, groupCode string) (bool, error) {
				return false, nil
			},
			IsGroupStageFinishedFunc: func(ctx context.Context) (bool, error) {
				return true, nil
			},
		}

		gss := &mocks.MockGroupStandingService{
			RecalculateStandingsFunc: func(ctx context.Context) error { return nil },
		}

		ss := &mocks.MockScoringService{
			ScoreBestThirdsFunc: func(ctx context.Context) ([]string, error) { return nil, nil },
		}

		gsr := &mocks.MockGroupStandingRepository{
			GetThirdPlaceGroupsFunc: func(ctx context.Context) ([]*domain.ThirdPlaceTeam, error) {
				return conflictTeams, nil
			},
		}

		service := newTestMatchService(mr, gsr, gss, ss, &mocks.MockCompetitionScoringService{
			RecomputeForMatchesFunc:    func(ctx context.Context, result *domain.ScoreMatchesResult) error { return nil },
		}, nil)

		// Select POR (group I) over NED (group A) from the tied pair — both are valid candidates.
		outcomes, err := service.ResolveThirdPlaceConflict(context.Background(), dtos.ResolveThirdPlaceConflictDto{
			TeamFifaCodes: []string{"BRA", "GER", "FRA", "ESP", "ARG", "MEX", "ENG", "POR"},
		})

		assert.NoError(t, err)
		assert.True(t, outcomes.IsGroupStageFinished)
	})

	t.Run("returns ErrThirdPlaceNotInConflict when standings have no tie", func(t *testing.T) {
		t.Parallel()

		gsr := &mocks.MockGroupStandingRepository{
			GetThirdPlaceGroupsFunc: func(ctx context.Context) ([]*domain.ThirdPlaceTeam, error) {
				return thirdPlaceTeamsNoConflict(), nil
			},
		}

		service := newTestMatchService(nil, gsr, nil, nil, nil, nil)

		outcomes, err := service.ResolveThirdPlaceConflict(context.Background(), dtos.ResolveThirdPlaceConflictDto{
			TeamFifaCodes: []string{"BRA", "GER", "FRA", "ESP", "ARG", "MEX", "USA", "CAN"},
		})

		assert.ErrorIs(t, err, domain.ErrThirdPlaceNotInConflict)
		assert.Nil(t, outcomes)
	})

	t.Run("returns ErrThirdPlaceInvalidSelection when a code is not in the candidate list", func(t *testing.T) {
		t.Parallel()

		gsr := &mocks.MockGroupStandingRepository{
			GetThirdPlaceGroupsFunc: func(ctx context.Context) ([]*domain.ThirdPlaceTeam, error) {
				return thirdPlaceTeamsWithConflict(), nil
			},
		}

		service := newTestMatchService(nil, gsr, nil, nil, nil, nil)

		// QAT is not among the 12 third-place teams
		outcomes, err := service.ResolveThirdPlaceConflict(context.Background(), dtos.ResolveThirdPlaceConflictDto{
			TeamFifaCodes: []string{"BRA", "GER", "FRA", "ESP", "ARG", "MEX", "ENG", "QAT"},
		})

		assert.ErrorIs(t, err, domain.ErrThirdPlaceInvalidSelection)
		assert.Nil(t, outcomes)
	})

	t.Run("returns ErrThirdPlaceInvalidSelection for duplicate codes", func(t *testing.T) {
		t.Parallel()

		service := newTestMatchService(nil, nil, nil, nil, nil, nil)

		outcomes, err := service.ResolveThirdPlaceConflict(context.Background(), dtos.ResolveThirdPlaceConflictDto{
			TeamFifaCodes: []string{"BRA", "GER", "FRA", "ESP", "ARG", "MEX", "ENG", "BRA"},
		})

		assert.ErrorIs(t, err, domain.ErrThirdPlaceInvalidSelection)
		assert.Nil(t, outcomes)
	})

	t.Run("returns ErrThirdPlaceInvalidSelection when payload does not contain exactly 8 codes", func(t *testing.T) {
		t.Parallel()

		service := newTestMatchService(nil, nil, nil, nil, nil, nil)

		outcomes, err := service.ResolveThirdPlaceConflict(context.Background(), dtos.ResolveThirdPlaceConflictDto{
			TeamFifaCodes: []string{"BRA", "GER", "FRA", "ESP", "ARG", "MEX", "ENG"},
		})

		assert.ErrorIs(t, err, domain.ErrThirdPlaceInvalidSelection)
		assert.Nil(t, outcomes)
	})
}

// ---------------------------------------------------------------------------
// TestMatchService_AdvanceBracket
// ---------------------------------------------------------------------------
func TestMatchService_AdvanceBracket(t *testing.T) {
	t.Parallel()

	t.Run("advances a knockout winner into the downstream slot, leaving the undecided side untouched", func(t *testing.T) {
		t.Parallel()

		var captured []domain.MatchTeamUpdate
		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{knockoutMatch73CanBeatRsa()}, nil
			},
			UpdateMatchTeamsFunc: func(ctx context.Context, updates []domain.MatchTeamUpdate) error {
				captured = updates
				return nil
			},
		}
		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		err := service.AdvanceBracket(context.Background(), []int64{73})

		assert.NoError(t, err)
		assert.Len(t, captured, 1)
		assert.Equal(t, int64(90), captured[0].MatchID)
		assert.Equal(t, "CAN", *captured[0].HomeTeamFifaCode)
		assert.Nil(t, captured[0].AwayTeamFifaCode) // winner of match 75 not known yet
	})

	t.Run("resolves losers into the third-place match and winners into the final", func(t *testing.T) {
		t.Parallel()

		winner101, winner102 := "FRA", "ARG"
		semiFinal101 := &domain.Match{
			ID: 101, StageCode: domain.MatchStageCodeSemiFinals, Status: domain.MatchStatusFinished,
			Teams:  domain.MatchTeams{Home: &domain.Team{FifaCode: "FRA"}, Away: &domain.Team{FifaCode: "BRA"}},
			Result: &domain.MatchResult{HomeScore: 2, AwayScore: 1, WinnerTeamFifaCode: &winner101},
		}
		semiFinal102 := &domain.Match{
			ID: 102, StageCode: domain.MatchStageCodeSemiFinals, Status: domain.MatchStatusFinished,
			Teams:  domain.MatchTeams{Home: &domain.Team{FifaCode: "ARG"}, Away: &domain.Team{FifaCode: "GER"}},
			Result: &domain.MatchResult{HomeScore: 3, AwayScore: 0, WinnerTeamFifaCode: &winner102},
		}

		var captured []domain.MatchTeamUpdate
		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{semiFinal101, semiFinal102}, nil
			},
			UpdateMatchTeamsFunc: func(ctx context.Context, updates []domain.MatchTeamUpdate) error {
				captured = updates
				return nil
			},
		}
		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		err := service.AdvanceBracket(context.Background(), []int64{101, 102})

		assert.NoError(t, err)
		byID := make(map[int64]domain.MatchTeamUpdate, len(captured))
		for _, update := range captured {
			byID[update.MatchID] = update
		}
		assert.Equal(t, "BRA", *byID[103].HomeTeamFifaCode) // loser of semifinal 101
		assert.Equal(t, "GER", *byID[103].AwayTeamFifaCode) // loser of semifinal 102
		assert.Equal(t, "FRA", *byID[104].HomeTeamFifaCode) // winner of semifinal 101
		assert.Equal(t, "ARG", *byID[104].AwayTeamFifaCode) // winner of semifinal 102
	})

	t.Run("is idempotent across repeated runs", func(t *testing.T) {
		t.Parallel()

		var runs [][]domain.MatchTeamUpdate
		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{knockoutMatch73CanBeatRsa()}, nil
			},
			UpdateMatchTeamsFunc: func(ctx context.Context, updates []domain.MatchTeamUpdate) error {
				runs = append(runs, updates)
				return nil
			},
		}
		service := newTestMatchService(mr, nil, nil, nil, nil, nil)

		assert.NoError(t, service.AdvanceBracket(context.Background(), []int64{73}))
		assert.NoError(t, service.AdvanceBracket(context.Background(), []int64{73}))
		assert.Len(t, runs, 2)
		assert.Equal(t, runs[0], runs[1])
	})

	t.Run("is a no-op for empty input and never touches the repository", func(t *testing.T) {
		t.Parallel()

		// All repository funcs are nil, so any repository call would panic.
		service := newTestMatchService(&mocks.MockMatchRepository{}, nil, nil, nil, nil, nil)

		assert.NoError(t, service.AdvanceBracket(context.Background(), nil))
	})
}
