package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockGroupStandingRepository struct {
	GetGroupStandingsFunc    func(ctx context.Context, groupCodes []string, position *int64) ([]*domain.GroupStanding, error)
	UpdateGroupStandingsFunc func(ctx context.Context, standings []*domain.GroupStanding) error
	GetThirdPlaceGroupsFunc  func(ctx context.Context) ([]*domain.ThirdPlaceTeam, error)
}

func (m *MockGroupStandingRepository) GetGroupStandings(ctx context.Context, groupCodes []string, position *int64) ([]*domain.GroupStanding, error) {
	if m.GetGroupStandingsFunc != nil {
		return m.GetGroupStandingsFunc(ctx, groupCodes, position)
	}
	panic("GetGroupStandings called unexpectedly")
}

func (m *MockGroupStandingRepository) UpdateGroupStandings(ctx context.Context, standings []*domain.GroupStanding) error {
	if m.UpdateGroupStandingsFunc != nil {
		return m.UpdateGroupStandingsFunc(ctx, standings)
	}
	panic("UpdateGroupStandings called unexpectedly")
}

func (m *MockGroupStandingRepository) GetThirdPlaceGroups(ctx context.Context) ([]*domain.ThirdPlaceTeam, error) {
	if m.GetThirdPlaceGroupsFunc != nil {
		return m.GetThirdPlaceGroupsFunc(ctx)
	}
	panic("GetThirdPlaceGroups called unexpectedly")
}
