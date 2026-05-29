package services

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type PlayerServiceInterface interface {
	SearchPlayers(ctx context.Context, filters domain.PlayerSearchFilters, page, limit int) (*domain.PlayerPage, error)
	GetPlayersByIDs(ctx context.Context, ids []int64) ([]*domain.Player, error)
}

type PlayerService struct {
	playerRepository domain.PlayerRepository
}

func NewPlayerService(playerRepository domain.PlayerRepository) PlayerServiceInterface {
	return &PlayerService{playerRepository: playerRepository}
}

func (s *PlayerService) SearchPlayers(
	ctx context.Context,
	filters domain.PlayerSearchFilters,
	page, limit int,
) (*domain.PlayerPage, error) {
	return s.playerRepository.SearchPlayers(ctx, filters, page, limit)
}

func (s *PlayerService) GetPlayersByIDs(ctx context.Context, ids []int64) ([]*domain.Player, error) {
	return s.playerRepository.GetPlayersByIDs(ctx, ids)
}
