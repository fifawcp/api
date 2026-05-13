package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockCompetitionScoreRepository struct {
	FindMatchCompetitionsByMatchesFunc func(ctx context.Context, matchIDs []int64) ([]int64, error)
	BatchUpsertMatchScoresFunc         func(ctx context.Context, competitionID int64, userIDs []string, exactScorePts int) error
	BatchUpsertPickemScoresFunc        func(ctx context.Context, competitionIDs []int64, userIDs []string) error
	GetLeaderboardFunc                 func(ctx context.Context, competitionID int64, page, limit int) (*domain.CompetitionLeaderboardPage, error)
}

func (m *MockCompetitionScoreRepository) FindMatchCompetitionsByMatches(ctx context.Context, matchIDs []int64) ([]int64, error) {
	if m.FindMatchCompetitionsByMatchesFunc != nil {
		return m.FindMatchCompetitionsByMatchesFunc(ctx, matchIDs)
	}
	panic("FindMatchCompetitionsByMatches called unexpectedly")
}

func (m *MockCompetitionScoreRepository) BatchUpsertMatchScores(ctx context.Context, competitionID int64, userIDs []string, exactScorePts int) error {
	if m.BatchUpsertMatchScoresFunc != nil {
		return m.BatchUpsertMatchScoresFunc(ctx, competitionID, userIDs, exactScorePts)
	}
	panic("BatchUpsertMatchScores called unexpectedly")
}

func (m *MockCompetitionScoreRepository) BatchUpsertPickemScores(ctx context.Context, competitionIDs []int64, userIDs []string) error {
	if m.BatchUpsertPickemScoresFunc != nil {
		return m.BatchUpsertPickemScoresFunc(ctx, competitionIDs, userIDs)
	}
	panic("BatchUpsertPickemScores called unexpectedly")
}

func (m *MockCompetitionScoreRepository) GetLeaderboard(ctx context.Context, competitionID int64, page, limit int) (*domain.CompetitionLeaderboardPage, error) {
	if m.GetLeaderboardFunc != nil {
		return m.GetLeaderboardFunc(ctx, competitionID, page, limit)
	}
	panic("GetLeaderboard called unexpectedly")
}
