package services

import (
	"context"

	"github.com/ncondes/fifawcp/internal/domain"
)

type BoardRankingServiceInterface interface {
	GetBoardRanking(ctx context.Context, boardID string) ([]*domain.BoardRanking, error)
}

type BoardRankingService struct {
	boardRankingRepository domain.BoardRankingRepository
}

func NewBoardRankingService(
	boardRankingRepository domain.BoardRankingRepository,
) BoardRankingServiceInterface {
	return &BoardRankingService{
		boardRankingRepository: boardRankingRepository,
	}
}

func (s *BoardRankingService) GetBoardRanking(ctx context.Context, boardID string) ([]*domain.BoardRanking, error) {
	return s.boardRankingRepository.GetBoardRanking(ctx, boardID)
}
