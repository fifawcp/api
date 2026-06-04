package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockCompetitionScoreRepository struct {
	FindMatchCompetitionsByMatchesFunc func(ctx context.Context, matchIDs []int64) ([]int64, error)
	FindPoolCompetitionsByMatchesFunc  func(ctx context.Context, matchIDs []int64) ([]int64, error)
	BatchUpsertMatchScoresFunc         func(ctx context.Context, competitionID int64, userIDs []string, exactScorePts int) error
	BatchUpsertPoolScoresFunc          func(ctx context.Context, competitionID int64, userIDs []string, exactScorePts int) error
	BatchUpsertPickemScoresFunc        func(ctx context.Context, competitionIDs []int64, userIDs []string) error
	GetLeaderboardFunc                 func(ctx context.Context, competitionID int64, page, limit int, q string) (*domain.CompetitionLeaderboardPage, error)
	GetUserPickemStatsFunc             func(ctx context.Context, competitionID int64, userID string) (domain.CompetitionUserStats, error)
	GetUserMatchStatsFunc              func(ctx context.Context, competitionID int64, userID string) (domain.CompetitionUserStats, error)
}

func (m *MockCompetitionScoreRepository) FindMatchCompetitionsByMatches(ctx context.Context, matchIDs []int64) ([]int64, error) {
	if m.FindMatchCompetitionsByMatchesFunc != nil {
		return m.FindMatchCompetitionsByMatchesFunc(ctx, matchIDs)
	}
	panic("FindMatchCompetitionsByMatches called unexpectedly")
}

func (m *MockCompetitionScoreRepository) FindPoolCompetitionsByMatches(ctx context.Context, matchIDs []int64) ([]int64, error) {
	if m.FindPoolCompetitionsByMatchesFunc != nil {
		return m.FindPoolCompetitionsByMatchesFunc(ctx, matchIDs)
	}
	panic("FindPoolCompetitionsByMatches called unexpectedly")
}

func (m *MockCompetitionScoreRepository) BatchUpsertMatchScores(ctx context.Context, competitionID int64, userIDs []string, exactScorePts int) error {
	if m.BatchUpsertMatchScoresFunc != nil {
		return m.BatchUpsertMatchScoresFunc(ctx, competitionID, userIDs, exactScorePts)
	}
	panic("BatchUpsertMatchScores called unexpectedly")
}

func (m *MockCompetitionScoreRepository) BatchUpsertPoolScores(ctx context.Context, competitionID int64, userIDs []string, exactScorePts int) error {
	if m.BatchUpsertPoolScoresFunc != nil {
		return m.BatchUpsertPoolScoresFunc(ctx, competitionID, userIDs, exactScorePts)
	}
	panic("BatchUpsertPoolScores called unexpectedly")
}

func (m *MockCompetitionScoreRepository) BatchUpsertPickemScores(ctx context.Context, competitionIDs []int64, userIDs []string) error {
	if m.BatchUpsertPickemScoresFunc != nil {
		return m.BatchUpsertPickemScoresFunc(ctx, competitionIDs, userIDs)
	}
	panic("BatchUpsertPickemScores called unexpectedly")
}

func (m *MockCompetitionScoreRepository) GetLeaderboard(ctx context.Context, competitionID int64, page, limit int, q string) (*domain.CompetitionLeaderboardPage, error) {
	if m.GetLeaderboardFunc != nil {
		return m.GetLeaderboardFunc(ctx, competitionID, page, limit, q)
	}
	panic("GetLeaderboard called unexpectedly")
}

func (m *MockCompetitionScoreRepository) GetUserPickemStats(ctx context.Context, competitionID int64, userID string) (domain.CompetitionUserStats, error) {
	if m.GetUserPickemStatsFunc != nil {
		return m.GetUserPickemStatsFunc(ctx, competitionID, userID)
	}
	panic("GetUserPickemStats called unexpectedly")
}

func (m *MockCompetitionScoreRepository) GetUserMatchStats(ctx context.Context, competitionID int64, userID string) (domain.CompetitionUserStats, error) {
	if m.GetUserMatchStatsFunc != nil {
		return m.GetUserMatchStatsFunc(ctx, competitionID, userID)
	}
	panic("GetUserMatchStats called unexpectedly")
}
