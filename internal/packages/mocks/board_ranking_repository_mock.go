package mocks

import (
	"context"

	"github.com/ncondes/fifawcp/internal/domain"
)

type MockBoardRankingRepository struct {
	GetBoardRankingFunc func(ctx context.Context, boardID string) ([]*domain.BoardRanking, error)
}

func (m *MockBoardRankingRepository) GetBoardRanking(ctx context.Context, boardID string) ([]*domain.BoardRanking, error) {
	if m.GetBoardRankingFunc != nil {
		return m.GetBoardRankingFunc(ctx, boardID)
	}
	panic("GetBoardRanking called unexpectedly")
}
