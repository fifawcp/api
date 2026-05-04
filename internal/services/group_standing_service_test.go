package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

func newTestGroupStandingService(
	gr *mocks.MockGroupStandingRepository,
	mr *mocks.MockMatchRepository,
	logger *mocks.MockLogger,
) GroupStandingServiceInterface {
	return NewGroupStandingService(gr, mr, logger)
}

func intScorePtr(n int) *int {
	v := n
	return &v
}

func stringPtr(s string) *string {
	v := s
	return &v
}

// ---------------------------------------------------------------------------
// TestGroupStandingService_GetGroupStandings
// ---------------------------------------------------------------------------
func TestGroupStandingService_GetGroupStandings(t *testing.T) {
	t.Parallel()

	t.Run("returns group standings", func(t *testing.T) {
		t.Parallel()

		gr := &mocks.MockGroupStandingRepository{
			GetGroupStandingsFunc: func(ctx context.Context, groupCodes []string, position *int64) ([]*domain.GroupStanding, error) {
				return []*domain.GroupStanding{
					{
						Position: 1,
						Team:     domain.Team{FifaCode: "GER"},
					},
				}, nil
			},
		}

		service := newTestGroupStandingService(gr, nil, nil)

		groupStandings, err := service.GetGroupStandings(context.Background(), []string{"A"}, nil)

		assert.NoError(t, err)
		assert.Equal(t, groupStandings, groupStandings)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		gr := &mocks.MockGroupStandingRepository{
			GetGroupStandingsFunc: func(ctx context.Context, groupCodes []string, position *int64) ([]*domain.GroupStanding, error) {
				return nil, errors.New("database error")
			},
		}

		service := newTestGroupStandingService(gr, nil, nil)

		groupStandings, err := service.GetGroupStandings(context.Background(), []string{"A"}, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, groupStandings)
	})
}

// ---------------------------------------------------------------------------
// TestGroupStandingService_RecalculateStandings
// ---------------------------------------------------------------------------
func TestGroupStandingService_RecalculateStandings(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on success", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				rows := []struct {
					id                   int64
					groupCode            string
					homeFifa, awayFifa   string
					homeScore, awayScore int
				}{
					{1, "A", "MEX", "RSA", 2, 1},
					{2, "A", "KOR", "CZE", 0, 3},
					{25, "A", "CZE", "RSA", 4, 0},
					{28, "A", "MEX", "KOR", 1, 2},
					{53, "A", "RSA", "KOR", 0, 1},
					{54, "A", "CZE", "MEX", 2, 2},
				}

				now := time.Now()
				matches := make([]*domain.Match, len(rows))
				for i, r := range rows {
					matches[i] = &domain.Match{
						ID:        r.id,
						GroupCode: stringPtr(r.groupCode),
						Status:    domain.MatchStatusFinished,
						StageCode: domain.MatchStageCodeGroupStage,
						HomeTeam:  &domain.Team{FifaCode: r.homeFifa},
						AwayTeam:  &domain.Team{FifaCode: r.awayFifa},
						HomeScore: intScorePtr(r.homeScore),
						AwayScore: intScorePtr(r.awayScore),
						UpdatedAt: now,
					}
				}

				return matches, nil
			},
		}

		gr := &mocks.MockGroupStandingRepository{
			UpdateGroupStandingsFunc: func(ctx context.Context, standings []*domain.GroupStanding) error {
				return nil
			},
		}

		service := newTestGroupStandingService(gr, mr, nil)

		err := service.RecalculateStandings(context.Background())

		assert.NoError(t, err)
	})

	t.Run("propagates match repository error", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return nil, errors.New("database error")
			},
		}

		service := newTestGroupStandingService(nil, mr, nil)

		err := service.RecalculateStandings(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})

	t.Run("propagates group standing repository error when recalculating standings by group", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return nil, nil
			},
		}

		gr := &mocks.MockGroupStandingRepository{
			UpdateGroupStandingsFunc: func(ctx context.Context, standings []*domain.GroupStanding) error {
				return errors.New("database error")
			},
		}

		service := newTestGroupStandingService(gr, mr, nil)

		err := service.RecalculateStandings(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

// ---------------------------------------------------------------------------
// TestGroupStandingService_rankGroup
// ---------------------------------------------------------------------------
func TestGroupStandingService_rankGroup(t *testing.T) {
	t.Parallel()

	type matchScore struct {
		home, away           string
		homeScore, awayScore int
	}

	buildMatches := func(scores []matchScore) []*domain.Match {
		matches := make([]*domain.Match, len(scores))
		for i, s := range scores {
			matches[i] = &domain.Match{
				HomeTeam:  &domain.Team{FifaCode: s.home},
				AwayTeam:  &domain.Team{FifaCode: s.away},
				HomeScore: intScorePtr(s.homeScore),
				AwayScore: intScorePtr(s.awayScore),
			}
		}

		return matches
	}

	testCases := []struct {
		name          string
		scores        []matchScore
		expectedOrder []string
	}{
		{
			name: "mid-tournament: tied teams haven't played each other yet",
			scores: []matchScore{
				{"MEX", "RSA", 1, 0},
				{"KOR", "CZE", 1, 0},
			},
			expectedOrder: []string{"MEX", "KOR", "CZE", "RSA"},
		},
		{
			name: "no ties",
			scores: []matchScore{
				{"MEX", "RSA", 0, 2},
				{"KOR", "CZE", 1, 0},
				{"CZE", "RSA", 0, 1},
				{"MEX", "KOR", 1, 1},
				{"RSA", "KOR", 2, 0},
				{"CZE", "MEX", 3, 0},
			},
			expectedOrder: []string{"RSA", "KOR", "CZE", "MEX"},
		},
		{
			name: "2 teams tied on points and goal difference",
			scores: []matchScore{
				{"MEX", "RSA", 0, 2},
				{"KOR", "CZE", 0, 0},
				{"CZE", "RSA", 1, 0},
				{"MEX", "KOR", 0, 1},
				{"RSA", "KOR", 1, 1},
				{"CZE", "MEX", 2, 0},
			},
			expectedOrder: []string{"CZE", "KOR", "RSA", "MEX"},
		},
		{
			name: "3 teams tied on points",
			scores: []matchScore{
				{"MEX", "RSA", 0, 1},
				{"KOR", "CZE", 1, 0},
				{"CZE", "RSA", 1, 0},
				{"MEX", "KOR", 0, 2},
				{"RSA", "KOR", 1, 0},
				{"CZE", "MEX", 3, 0},
			},
			expectedOrder: []string{"CZE", "KOR", "RSA", "MEX"},
		},
		{
			name: "3 teams tied on points and goal difference",
			scores: []matchScore{
				{"MEX", "RSA", 0, 2},
				{"KOR", "CZE", 0, 0},
				{"CZE", "RSA", 1, 1},
				{"MEX", "KOR", 0, 2},
				{"RSA", "KOR", 2, 2},
				{"CZE", "MEX", 2, 0},
			},
			expectedOrder: []string{"RSA", "KOR", "CZE", "MEX"},
		},
		{
			name: "4 teams tied on points",
			scores: []matchScore{
				{"MEX", "RSA", 1, 1},
				{"KOR", "CZE", 1, 1},
				{"CZE", "RSA", 0, 1},
				{"MEX", "KOR", 1, 0},
				{"RSA", "KOR", 0, 1},
				{"CZE", "MEX", 2, 0},
			},
			expectedOrder: []string{"CZE", "KOR", "RSA", "MEX"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			standings := rankGroup(buildMatches(tc.scores))

			actualOrder := make([]string, len(standings))
			for i, s := range standings {
				actualOrder[i] = s.Team.FifaCode
			}

			// ? Debug output
			// for _, s := range standings {
			// 	if tc.name == "no ties" {
			// 		fmt.Printf("Team: %s, Points: %d, GD: %d, GF: %d, GA: %d\n", s.Team.FifaCode, s.Points, s.GoalDifference, s.GoalsFor, s.GoalsAgainst)
			// 	}
			// }

			assert.Equal(t, tc.expectedOrder, actualOrder)
		})
	}
}
