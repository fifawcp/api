package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockPlayerRepository struct {
	SearchPlayersFunc   func(ctx context.Context, filters domain.PlayerSearchFilters, page, limit int) (*domain.PlayerPage, error)
	GetPlayersByIDsFunc func(ctx context.Context, ids []int64) ([]*domain.Player, error)
	UpsertPlayersFunc   func(ctx context.Context, players []*domain.Player) error
}

func (m *MockPlayerRepository) SearchPlayers(ctx context.Context, filters domain.PlayerSearchFilters, page, limit int) (*domain.PlayerPage, error) {
	if m.SearchPlayersFunc != nil {
		return m.SearchPlayersFunc(ctx, filters, page, limit)
	}
	panic("SearchPlayers called unexpectedly")
}

func (m *MockPlayerRepository) GetPlayersByIDs(ctx context.Context, ids []int64) ([]*domain.Player, error) {
	if m.GetPlayersByIDsFunc != nil {
		return m.GetPlayersByIDsFunc(ctx, ids)
	}
	panic("GetPlayersByIDs called unexpectedly")
}

func (m *MockPlayerRepository) UpsertPlayers(ctx context.Context, players []*domain.Player) error {
	if m.UpsertPlayersFunc != nil {
		return m.UpsertPlayersFunc(ctx, players)
	}
	panic("UpsertPlayers called unexpectedly")
}
