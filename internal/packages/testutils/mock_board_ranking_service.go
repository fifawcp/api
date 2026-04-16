package testutils

import (
	"context"

	"github.com/ncondes/fifawcp/internal/domain"
)

type MockBoardRankingService struct {
	GetBoardRankingFunc func(ctx context.Context, boardID string) ([]*domain.BoardRanking, error)
}

func (m *MockBoardRankingService) GetBoardRanking(ctx context.Context, boardID string) ([]*domain.BoardRanking, error) {
	if m.GetBoardRankingFunc != nil {
		return m.GetBoardRankingFunc(ctx, boardID)
	}
	panic("GetBoardRanking called unexpectedly")
}
