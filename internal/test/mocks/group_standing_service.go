package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockGroupStandingService struct {
	GetGroupStandingsFunc    func(ctx context.Context, groupCodes []string, position *int64) ([]*domain.GroupStanding, error)
	RecalculateStandingsFunc func(ctx context.Context) error
}

func (m *MockGroupStandingService) GetGroupStandings(ctx context.Context, groupCodes []string, position *int64) ([]*domain.GroupStanding, error) {
	if m.GetGroupStandingsFunc != nil {
		return m.GetGroupStandingsFunc(ctx, groupCodes, position)
	}
	panic("GetGroupStandings called unexpectedly")
}

func (m *MockGroupStandingService) RecalculateStandings(ctx context.Context) error {
	if m.RecalculateStandingsFunc != nil {
		return m.RecalculateStandingsFunc(ctx)
	}
	panic("RecalculateStandings called unexpectedly")
}
