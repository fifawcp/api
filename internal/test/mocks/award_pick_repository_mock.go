package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockAwardPickRepository struct {
	GetAwardPicksFunc         func(ctx context.Context, userID string) ([]*domain.UserAwardPick, error)
	UpsertAwardPicksFunc      func(ctx context.Context, userID string, picks []*domain.UserAwardPick) error
	GetAwardPicksByPlayerFunc func(ctx context.Context, awardType domain.AwardType, playerID int64) ([]*domain.UserAwardPick, error)
	UpsertAwardWinnersFunc    func(ctx context.Context, winners []*domain.AwardWinner) error
	GetAwardWinnersFunc       func(ctx context.Context) ([]*domain.AwardWinner, error)
}

func (m *MockAwardPickRepository) GetAwardPicks(ctx context.Context, userID string) ([]*domain.UserAwardPick, error) {
	if m.GetAwardPicksFunc != nil {
		return m.GetAwardPicksFunc(ctx, userID)
	}
	panic("GetAwardPicks called unexpectedly")
}

func (m *MockAwardPickRepository) UpsertAwardPicks(ctx context.Context, userID string, picks []*domain.UserAwardPick) error {
	if m.UpsertAwardPicksFunc != nil {
		return m.UpsertAwardPicksFunc(ctx, userID, picks)
	}
	panic("UpsertAwardPicks called unexpectedly")
}

func (m *MockAwardPickRepository) GetAwardPicksByPlayer(ctx context.Context, awardType domain.AwardType, playerID int64) ([]*domain.UserAwardPick, error) {
	if m.GetAwardPicksByPlayerFunc != nil {
		return m.GetAwardPicksByPlayerFunc(ctx, awardType, playerID)
	}
	panic("GetAwardPicksByPlayer called unexpectedly")
}

func (m *MockAwardPickRepository) UpsertAwardWinners(ctx context.Context, winners []*domain.AwardWinner) error {
	if m.UpsertAwardWinnersFunc != nil {
		return m.UpsertAwardWinnersFunc(ctx, winners)
	}
	panic("UpsertAwardWinners called unexpectedly")
}

func (m *MockAwardPickRepository) GetAwardWinners(ctx context.Context) ([]*domain.AwardWinner, error) {
	if m.GetAwardWinnersFunc != nil {
		return m.GetAwardWinnersFunc(ctx)
	}
	panic("GetAwardWinners called unexpectedly")
}
