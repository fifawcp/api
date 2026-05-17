package services

import (
	"context"
	"testing"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

func newTestDashboardService(
	pickemService PickemServiceInterface,
	matchScorePickRepo domain.MatchScorePickRepository,
	matchRepository domain.MatchRepository,
	competitionScoreRepo domain.CompetitionScoreRepository,
	globalPickemCompetition *domain.Competition,
	globalMatchCompetition *domain.Competition,
) DashboardServiceInterface {
	return NewDashboardService(
		pickemService,
		matchScorePickRepo,
		matchRepository,
		competitionScoreRepo,
		globalPickemCompetition,
		globalMatchCompetition,
	)
}

func makeGlobalCompetitions() (*domain.Competition, *domain.Competition) {
	return &domain.Competition{
			ID:   11,
			Name: "Pick'em",
			Type: domain.CompetitionTypePickem,
		},
		&domain.Competition{
			ID:   12,
			Name: "All Matches",
			Type: domain.CompetitionTypeMatch,
		}
}

// ---------------------------------------------------------------------------
// TestDashboardService_GetDashboard
// ---------------------------------------------------------------------------
func TestDashboardService_GetDashboard(t *testing.T) {
	t.Parallel()

	t.Run("returns full dashboard with champion when bracket is complete", func(t *testing.T) {
		t.Parallel()

		argentina := &domain.Team{FifaCode: "ARG", Name: domain.TeamNames{"en": "Argentina"}}
		pickemSvc := &mocks.MockPickemService{
			GetChampionPickFunc: func(ctx context.Context, userID string) (*domain.Team, error) {
				return argentina, nil
			},
			GetUserPickemProgressFunc: func(ctx context.Context, userID string) (*domain.PickemProgress, error) {
				return &domain.PickemProgress{
					Groups:     domain.StepProgress{Completed: 12, Total: 12},
					BestThirds: domain.StepProgress{Completed: 8, Total: 8},
					Bracket:    domain.StepProgress{Completed: 32, Total: 32},
				}, nil
			},
		}

		competitionScoreRepo := &mocks.MockCompetitionScoreRepository{
			GetUserPickemStatsFunc: func(ctx context.Context, competitionID int64, userID string) (domain.CompetitionUserStats, error) {
				return domain.CompetitionUserStats{Rank: 3, Points: 150}, nil
			},
			GetUserMatchStatsFunc: func(ctx context.Context, competitionID int64, userID string) (domain.CompetitionUserStats, error) {
				return domain.CompetitionUserStats{Rank: 5, Points: 90}, nil
			},
			GetLeaderboardFunc: func(ctx context.Context, competitionID int64, page, limit int) (*domain.CompetitionLeaderboardPage, error) {
				assert.Equal(t, 1, page)
				assert.Equal(t, 5, limit)
				if competitionID == 11 {
					return &domain.CompetitionLeaderboardPage{
						Members: []*domain.CompetitionLeaderboardEntry{
							{Rank: 1, Member: domain.CompetitionLeaderboardMember{UserID: "u1", UserName: "alice"}, Score: &domain.PickemScore{Total: 100}},
						},
					}, nil
				}
				return &domain.CompetitionLeaderboardPage{
					Members: []*domain.CompetitionLeaderboardEntry{
						{Rank: 1, Member: domain.CompetitionLeaderboardMember{UserID: "u2", UserName: "bob"}, Score: &domain.MatchScore{Total: 75}},
					},
				}, nil
			},
		}

		kickoffTime := time.Date(2026, 6, 11, 18, 0, 0, 0, time.UTC)
		matchRepo := &mocks.MockMatchRepository{
			GetNextScheduledMatchFunc: func(ctx context.Context) (*domain.Match, error) {
				return &domain.Match{
					ID:        1,
					StageCode: domain.MatchStageCodeGroupStage,
					KickoffAt: kickoffTime,
					Teams: domain.MatchTeams{
						Home: &domain.Team{FifaCode: "MEX"},
						Away: &domain.Team{FifaCode: "RSA"},
					},
				}, nil
			},
		}

		matchScorePickRepo := &mocks.MockMatchScorePickRepository{
			CountMatchScorePicksByUserFunc: func(ctx context.Context, userID string) (int, error) {
				return 12, nil
			},
		}

		pickemComp, matchComp := makeGlobalCompetitions()
		service := newTestDashboardService(pickemSvc, matchScorePickRepo, matchRepo, competitionScoreRepo, pickemComp, matchComp)

		dashboard, err := service.GetDashboard(context.Background(), "user-1")

		assert.NoError(t, err)
		assert.NotNil(t, dashboard.PickedChampion)
		assert.Equal(t, "ARG", dashboard.PickedChampion.FifaCode)
		assert.NotNil(t, dashboard.Stats)
		assert.Equal(t, 3, dashboard.Stats.Pickem.Rank)
		assert.Equal(t, 150, dashboard.Stats.Pickem.Points)
		assert.Equal(t, 5, dashboard.Stats.Match.Rank)
		assert.Equal(t, 90, dashboard.Stats.Match.Points)
		assert.NotNil(t, dashboard.NextMatch)
		assert.Equal(t, int64(1), dashboard.NextMatch.ID)
		assert.Equal(t, "MEX", dashboard.NextMatch.Teams.Home.FifaCode)
		assert.Equal(t, "RSA", dashboard.NextMatch.Teams.Away.FifaCode)
		assert.Equal(t, kickoffTime, dashboard.NextMatch.KickoffAt)
		assert.Equal(t, 12, dashboard.Progress.MatchPicks.Completed)
		assert.Equal(t, 104, dashboard.Progress.MatchPicks.Total)
		assert.True(t, dashboard.Progress.Pickem.Groups.IsComplete())
		assert.True(t, dashboard.Progress.Pickem.BestThirds.IsComplete())
		assert.True(t, dashboard.Progress.Pickem.Bracket.IsComplete())
		assert.Len(t, dashboard.Leaderboard.Pickem.Entries, 1)
		assert.Equal(t, "Pick'em", dashboard.Leaderboard.Pickem.CompetitionName)
		assert.Equal(t, 100, dashboard.Leaderboard.Pickem.Entries[0].Points)
		assert.Equal(t, "All Matches", dashboard.Leaderboard.Match.CompetitionName)
		assert.Len(t, dashboard.Leaderboard.Match.Entries, 1)
		assert.Equal(t, 75, dashboard.Leaderboard.Match.Entries[0].Points)
	})

	t.Run("returns nil champion when bracket is incomplete", func(t *testing.T) {
		t.Parallel()

		pickemSvc := &mocks.MockPickemService{
			GetChampionPickFunc: func(ctx context.Context, userID string) (*domain.Team, error) {
				return nil, nil
			},
			GetUserPickemProgressFunc: func(ctx context.Context, userID string) (*domain.PickemProgress, error) {
				return &domain.PickemProgress{
					Groups:     domain.StepProgress{Completed: 5, Total: 12},
					BestThirds: domain.StepProgress{Completed: 0, Total: 8},
					Bracket:    domain.StepProgress{Completed: 0, Total: 32},
				}, nil
			},
		}

		competitionScoreRepo := &mocks.MockCompetitionScoreRepository{
			GetUserPickemStatsFunc: func(ctx context.Context, competitionID int64, userID string) (domain.CompetitionUserStats, error) {
				return domain.CompetitionUserStats{}, nil
			},
			GetUserMatchStatsFunc: func(ctx context.Context, competitionID int64, userID string) (domain.CompetitionUserStats, error) {
				return domain.CompetitionUserStats{}, nil
			},
			GetLeaderboardFunc: func(ctx context.Context, competitionID int64, page, limit int) (*domain.CompetitionLeaderboardPage, error) {
				return &domain.CompetitionLeaderboardPage{Members: []*domain.CompetitionLeaderboardEntry{}}, nil
			},
		}

		matchRepo := &mocks.MockMatchRepository{
			GetNextScheduledMatchFunc: func(ctx context.Context) (*domain.Match, error) {
				return nil, nil
			},
		}

		matchScorePickRepo := &mocks.MockMatchScorePickRepository{
			CountMatchScorePicksByUserFunc: func(ctx context.Context, userID string) (int, error) {
				return 0, nil
			},
		}

		pickemComp, matchComp := makeGlobalCompetitions()
		service := newTestDashboardService(pickemSvc, matchScorePickRepo, matchRepo, competitionScoreRepo, pickemComp, matchComp)

		dashboard, err := service.GetDashboard(context.Background(), "user-1")

		assert.NoError(t, err)
		assert.Nil(t, dashboard.PickedChampion)
		assert.NotNil(t, dashboard.Stats)
		assert.Equal(t, 0, dashboard.Stats.Pickem.Rank)
		assert.Equal(t, 0, dashboard.Stats.Match.Rank)
		assert.Nil(t, dashboard.NextMatch)
		assert.NotNil(t, dashboard.Progress)
		assert.Equal(t, 0, dashboard.Progress.MatchPicks.Completed)
		assert.Equal(t, 104, dashboard.Progress.MatchPicks.Total)
		assert.Equal(t, 5, dashboard.Progress.Pickem.Groups.Completed)
		assert.False(t, dashboard.Progress.Pickem.Bracket.IsComplete())
		assert.Empty(t, dashboard.Leaderboard.Pickem.Entries)
		assert.Empty(t, dashboard.Leaderboard.Match.Entries)
	})

	t.Run("returns public-only dashboard for guest (empty userID)", func(t *testing.T) {
		t.Parallel()

		// Per-user mock funcs are left unset — they panic if called, proving the
		// service does not fan out user-specific queries for guests.
		pickemSvc := &mocks.MockPickemService{}
		matchScorePickRepo := &mocks.MockMatchScorePickRepository{}

		kickoffTime := time.Date(2026, 6, 11, 18, 0, 0, 0, time.UTC)
		matchRepo := &mocks.MockMatchRepository{
			GetNextScheduledMatchFunc: func(ctx context.Context) (*domain.Match, error) {
				return &domain.Match{ID: 1, KickoffAt: kickoffTime}, nil
			},
		}

		competitionScoreRepo := &mocks.MockCompetitionScoreRepository{
			GetLeaderboardFunc: func(ctx context.Context, competitionID int64, page, limit int) (*domain.CompetitionLeaderboardPage, error) {
				return &domain.CompetitionLeaderboardPage{
					Members: []*domain.CompetitionLeaderboardEntry{
						{Rank: 1, Member: domain.CompetitionLeaderboardMember{UserID: "u1", UserName: "alice"}, Score: &domain.PickemScore{Total: 100}},
					},
				}, nil
			},
		}

		pickemComp, matchComp := makeGlobalCompetitions()
		service := newTestDashboardService(pickemSvc, matchScorePickRepo, matchRepo, competitionScoreRepo, pickemComp, matchComp)

		dashboard, err := service.GetDashboard(context.Background(), "")

		assert.NoError(t, err)
		assert.Nil(t, dashboard.PickedChampion)
		assert.Nil(t, dashboard.Stats)
		assert.Nil(t, dashboard.Progress)
		assert.NotNil(t, dashboard.NextMatch)
		assert.Equal(t, int64(1), dashboard.NextMatch.ID)
		assert.Len(t, dashboard.Leaderboard.Pickem.Entries, 1)
		assert.Equal(t, "Pick'em", dashboard.Leaderboard.Pickem.CompetitionName)
		assert.Len(t, dashboard.Leaderboard.Match.Entries, 1)
		assert.Equal(t, "All Matches", dashboard.Leaderboard.Match.CompetitionName)
	})
}
