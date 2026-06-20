package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

func newTestGroupStandingService(
	gr *mocks.MockGroupStandingRepository,
	mr *mocks.MockMatchRepository,
	fp *mocks.MockMatchFairPlayRepository,
	logger *mocks.MockLogger,
) GroupStandingServiceInterface {
	return NewGroupStandingService(gr, mr, fp, logger)
}

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

		service := newTestGroupStandingService(gr, nil, nil, nil)

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

		service := newTestGroupStandingService(gr, nil, nil, nil)

		groupStandings, err := service.GetGroupStandings(context.Background(), []string{"A"}, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, groupStandings)
	})
}

func TestGroupStandingService_RecalculateStandings(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on success", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				rows := []struct {
					id                                 int64
					groupCode                          string
					homeTeamFifaCode, awayTeamFifaCode string
					homeScore, awayScore               int
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
						GroupCode: &r.groupCode,
						Status:    domain.MatchStatusFinished,
						StageCode: domain.MatchStageCodeGroupStage,
						Teams: domain.MatchTeams{
							Home: &domain.Team{FifaCode: r.homeTeamFifaCode},
							Away: &domain.Team{FifaCode: r.awayTeamFifaCode},
						},
						Result:    &domain.MatchResult{HomeScore: r.homeScore, AwayScore: r.awayScore},
						UpdatedAt: now,
					}
				}

				return matches, nil
			},
		}

		gr := &mocks.MockGroupStandingRepository{
			GetGroupStandingsFunc: func(_ context.Context, _ []string, _ *int64) ([]*domain.GroupStanding, error) {
				roster := []*domain.GroupStanding{}
				for _, code := range []string{"MEX", "RSA", "KOR", "CZE"} {
					roster = append(roster, &domain.GroupStanding{Team: domain.Team{FifaCode: code, GroupCode: "A"}})
				}
				return roster, nil
			},
			UpdateGroupStandingsFunc: func(ctx context.Context, standings []*domain.GroupStanding) error {
				return nil
			},
		}

		fp := &mocks.MockMatchFairPlayRepository{
			GetFairPlayTotalsByGroupFunc: func(_ context.Context, _ string) (map[string]int, error) {
				return map[string]int{}, nil
			},
		}

		service := newTestGroupStandingService(gr, mr, fp, nil)

		err := service.RecalculateStandings(context.Background())

		assert.NoError(t, err)
	})

	t.Run("applies fair play to break ties left level by head-to-head", func(t *testing.T) {
		t.Parallel()

		// Group G mirror: IRN drew NZL 2-2 and BEL drew EGY 1-1, so all four teams
		// sit on 1 point with 0 goal difference. IRN and NZL stay level through
		// head-to-head, so fair play decides — IRN took a yellow (-1), NZL none, so
		// NZL must rank above IRN even though IRN is higher in the FIFA world ranking.
		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(_ context.Context, _ domain.MatchFilters) ([]*domain.Match, error) {
				rows := []struct {
					id                                 int64
					homeTeamFifaCode, awayTeamFifaCode string
					homeScore, awayScore               int
				}{
					{16, "IRN", "NZL", 2, 2},
					{14, "BEL", "EGY", 1, 1},
				}

				groupCode := "G"
				matches := make([]*domain.Match, len(rows))
				for i, r := range rows {
					matches[i] = &domain.Match{
						ID:        r.id,
						GroupCode: &groupCode,
						Status:    domain.MatchStatusFinished,
						StageCode: domain.MatchStageCodeGroupStage,
						Teams: domain.MatchTeams{
							Home: &domain.Team{FifaCode: r.homeTeamFifaCode},
							Away: &domain.Team{FifaCode: r.awayTeamFifaCode},
						},
						Result: &domain.MatchResult{HomeScore: r.homeScore, AwayScore: r.awayScore},
					}
				}

				return matches, nil
			},
		}

		var mu sync.Mutex
		var groupGOrder []string
		gr := &mocks.MockGroupStandingRepository{
			GetGroupStandingsFunc: func(_ context.Context, _ []string, _ *int64) ([]*domain.GroupStanding, error) {
				roster := []*domain.GroupStanding{}
				for _, code := range []string{"IRN", "NZL", "BEL", "EGY"} {
					roster = append(roster, &domain.GroupStanding{Team: domain.Team{FifaCode: code, GroupCode: "G"}})
				}
				return roster, nil
			},
			UpdateGroupStandingsFunc: func(_ context.Context, standings []*domain.GroupStanding) error {
				if len(standings) == 0 {
					return nil
				}
				mu.Lock()
				defer mu.Unlock()
				groupGOrder = make([]string, len(standings))
				for i, s := range standings {
					groupGOrder[i] = s.Team.FifaCode
				}
				return nil
			},
		}

		fp := &mocks.MockMatchFairPlayRepository{
			GetFairPlayTotalsByGroupFunc: func(_ context.Context, groupCode string) (map[string]int, error) {
				if groupCode == "G" {
					return map[string]int{"IRN": -1}, nil
				}
				return map[string]int{}, nil
			},
		}

		service := newTestGroupStandingService(gr, mr, fp, nil)

		err := service.RecalculateStandings(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, []string{"NZL", "IRN", "BEL", "EGY"}, groupGOrder)
	})

	t.Run("propagates match repository error", func(t *testing.T) {
		t.Parallel()

		mr := &mocks.MockMatchRepository{
			GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
				return nil, errors.New("database error")
			},
		}

		service := newTestGroupStandingService(nil, mr, nil, nil)

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
			GetGroupStandingsFunc: func(_ context.Context, _ []string, _ *int64) ([]*domain.GroupStanding, error) {
				return []*domain.GroupStanding{}, nil
			},
			UpdateGroupStandingsFunc: func(ctx context.Context, standings []*domain.GroupStanding) error {
				return errors.New("database error")
			},
		}

		fp := &mocks.MockMatchFairPlayRepository{
			GetFairPlayTotalsByGroupFunc: func(_ context.Context, _ string) (map[string]int, error) {
				return map[string]int{}, nil
			},
		}

		service := newTestGroupStandingService(gr, mr, fp, nil)

		err := service.RecalculateStandings(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

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
				Teams: domain.MatchTeams{
					Home: &domain.Team{FifaCode: s.home},
					Away: &domain.Team{FifaCode: s.away},
				},
				Result: &domain.MatchResult{HomeScore: s.homeScore, AwayScore: s.awayScore},
			}
		}

		return matches
	}

	buildRoster := func(codes []string) []domain.Team {
		roster := make([]domain.Team, len(codes))
		for i, code := range codes {
			roster[i] = domain.Team{FifaCode: code}
		}

		return roster
	}

	groupRoster := []string{"MEX", "KOR", "CZE", "RSA"}

	testCases := []struct {
		name          string
		roster        []string
		scores        []matchScore
		fairPlay      map[string]int
		expectedOrder []string
	}{
		{
			// Regression: the screenshot bug. After one match only two of the four
			// teams have played; the other two must still be ranked (not dropped).
			name:   "early matchday: teams with no matches are still ranked",
			roster: groupRoster,
			scores: []matchScore{
				{"MEX", "RSA", 2, 0},
			},
			expectedOrder: []string{"MEX", "KOR", "CZE", "RSA"},
		},
		{
			name:   "mid-tournament: tied teams haven't played each other yet",
			roster: groupRoster,
			scores: []matchScore{
				{"MEX", "RSA", 1, 0},
				{"KOR", "CZE", 1, 0},
			},
			expectedOrder: []string{"MEX", "KOR", "CZE", "RSA"},
		},
		{
			name:   "no ties",
			roster: groupRoster,
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
			name:   "2 teams tied on points and goal difference",
			roster: groupRoster,
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
			name:   "3 teams tied on points",
			roster: groupRoster,
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
			name:   "3 teams tied on points and goal difference",
			roster: groupRoster,
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
			name:   "4 teams tied on points",
			roster: groupRoster,
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
		{
			// Level on points, goal difference, goals for and head-to-head, so fair
			// play (FIFA rule f) decides: MEX took a card, so RSA ranks above it even
			// though MEX is higher in the FIFA world ranking.
			name:   "fair play breaks a head-to-head tie",
			roster: []string{"MEX", "RSA"},
			scores: []matchScore{
				{"MEX", "RSA", 1, 1},
			},
			fairPlay:      map[string]int{"MEX": -1},
			expectedOrder: []string{"RSA", "MEX"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			standings := rankGroup(buildRoster(tc.roster), buildMatches(tc.scores), tc.fairPlay)

			actualOrder := make([]string, len(standings))
			for i, s := range standings {
				actualOrder[i] = s.Team.FifaCode
			}

			assert.Equal(t, tc.expectedOrder, actualOrder)
		})
	}
}
