package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockAwardService struct {
	GetUserAwardsFunc   func(ctx context.Context, userID string) (*domain.UserAwards, error)
	GetMemberAwardsFunc func(ctx context.Context, userID string) (*domain.UserAwards, error)
	SaveAwardPicksFunc  func(ctx context.Context, userID string, picks []*domain.UserAwardPick) (*domain.UserAwards, error)
	GetPopularPicksFunc func(ctx context.Context, limit int) (domain.PopularPicksByAward, error)
	RecordWinnersFunc   func(ctx context.Context, winners []*domain.AwardWinner) error
}

func (m *MockAwardService) GetUserAwards(ctx context.Context, userID string) (*domain.UserAwards, error) {
	if m.GetUserAwardsFunc != nil {
		return m.GetUserAwardsFunc(ctx, userID)
	}
	panic("GetUserAwards called unexpectedly")
}

func (m *MockAwardService) GetMemberAwards(ctx context.Context, userID string) (*domain.UserAwards, error) {
	if m.GetMemberAwardsFunc != nil {
		return m.GetMemberAwardsFunc(ctx, userID)
	}
	panic("GetMemberAwards called unexpectedly")
}

func (m *MockAwardService) SaveAwardPicks(ctx context.Context, userID string, picks []*domain.UserAwardPick) (*domain.UserAwards, error) {
	if m.SaveAwardPicksFunc != nil {
		return m.SaveAwardPicksFunc(ctx, userID, picks)
	}
	panic("SaveAwardPicks called unexpectedly")
}

func (m *MockAwardService) GetPopularPicks(ctx context.Context, limit int) (domain.PopularPicksByAward, error) {
	if m.GetPopularPicksFunc != nil {
		return m.GetPopularPicksFunc(ctx, limit)
	}
	panic("GetPopularPicks called unexpectedly")
}

func (m *MockAwardService) RecordWinners(ctx context.Context, winners []*domain.AwardWinner) error {
	if m.RecordWinnersFunc != nil {
		return m.RecordWinnersFunc(ctx, winners)
	}
	panic("RecordWinners called unexpectedly")
}
