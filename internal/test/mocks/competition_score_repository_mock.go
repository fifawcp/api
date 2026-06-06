package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockCompetitionScoreRepository struct {
	FindMatchCompetitionsByMatchesFunc func(ctx context.Context, matchIDs []int64) ([]int64, error)
	FindPickCompetitionsByMatchesFunc  func(ctx context.Context, matchIDs []int64) ([]int64, error)
	BatchUpsertMatchScoresFunc         func(ctx context.Context, competitionID int64, userIDs []string, exactScorePts int) error
	BatchUpsertPickScoresFunc          func(ctx context.Context, competitionID int64, userIDs []string, exactScorePts int) error
	GetLeaderboardFunc                 func(ctx context.Context, competitionID int64, page, limit int, q, sort, dir string) (*domain.CompetitionLeaderboardPage, error)
	GetBoardSummaryFunc                func(ctx context.Context, boardID int64, page, limit int, q, sort, dir string) (*domain.BoardSummaryPage, error)
	GetBoardCompetitionPreviewsFunc    func(ctx context.Context, boardID int64, limit int) (map[int64][]*domain.CompetitionLeaderboardEntry, error)
	GetUserPickemStatsFunc             func(ctx context.Context, competitionID int64, userID string) (domain.CompetitionUserStats, error)
	GetUserMatchStatsFunc              func(ctx context.Context, competitionID int64, userID string) (domain.CompetitionUserStats, error)
}

func (m *MockCompetitionScoreRepository) FindMatchCompetitionsByMatches(ctx context.Context, matchIDs []int64) ([]int64, error) {
	if m.FindMatchCompetitionsByMatchesFunc != nil {
		return m.FindMatchCompetitionsByMatchesFunc(ctx, matchIDs)
	}
	panic("FindMatchCompetitionsByMatches called unexpectedly")
}

func (m *MockCompetitionScoreRepository) FindPickCompetitionsByMatches(ctx context.Context, matchIDs []int64) ([]int64, error) {
	if m.FindPickCompetitionsByMatchesFunc != nil {
		return m.FindPickCompetitionsByMatchesFunc(ctx, matchIDs)
	}
	panic("FindPickCompetitionsByMatches called unexpectedly")
}

func (m *MockCompetitionScoreRepository) BatchUpsertMatchScores(ctx context.Context, competitionID int64, userIDs []string, exactScorePts int) error {
	if m.BatchUpsertMatchScoresFunc != nil {
		return m.BatchUpsertMatchScoresFunc(ctx, competitionID, userIDs, exactScorePts)
	}
	panic("BatchUpsertMatchScores called unexpectedly")
}

func (m *MockCompetitionScoreRepository) BatchUpsertPickScores(ctx context.Context, competitionID int64, userIDs []string, exactScorePts int) error {
	if m.BatchUpsertPickScoresFunc != nil {
		return m.BatchUpsertPickScoresFunc(ctx, competitionID, userIDs, exactScorePts)
	}
	panic("BatchUpsertPickScores called unexpectedly")
}

func (m *MockCompetitionScoreRepository) GetLeaderboard(ctx context.Context, competitionID int64, page, limit int, q, sort, dir string) (*domain.CompetitionLeaderboardPage, error) {
	if m.GetLeaderboardFunc != nil {
		return m.GetLeaderboardFunc(ctx, competitionID, page, limit, q, sort, dir)
	}
	panic("GetLeaderboard called unexpectedly")
}

func (m *MockCompetitionScoreRepository) GetBoardSummary(ctx context.Context, boardID int64, page, limit int, q, sort, dir string) (*domain.BoardSummaryPage, error) {
	if m.GetBoardSummaryFunc != nil {
		return m.GetBoardSummaryFunc(ctx, boardID, page, limit, q, sort, dir)
	}
	panic("GetBoardSummary called unexpectedly")
}

func (m *MockCompetitionScoreRepository) GetBoardCompetitionPreviews(ctx context.Context, boardID int64, limit int) (map[int64][]*domain.CompetitionLeaderboardEntry, error) {
	if m.GetBoardCompetitionPreviewsFunc != nil {
		return m.GetBoardCompetitionPreviewsFunc(ctx, boardID, limit)
	}
	panic("GetBoardCompetitionPreviews called unexpectedly")
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
