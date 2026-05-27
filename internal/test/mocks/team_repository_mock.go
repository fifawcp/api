package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockTeamRepository struct {
	GetAllTeamsFunc func(ctx context.Context) ([]*domain.Team, error)
}

func (m *MockTeamRepository) GetAllTeams(ctx context.Context) ([]*domain.Team, error) {
	if m.GetAllTeamsFunc != nil {
		return m.GetAllTeamsFunc(ctx)
	}
	panic("GetAllTeams called unexpectedly")
}
