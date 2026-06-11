package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/infrastructure/football"
)

type MockFixtureFetcher struct {
	GetFixtureFunc        func(ctx context.Context, fixtureID int64) (*football.FixtureResponse, error)
	GetFixturesByTeamFunc func(ctx context.Context, teamAPIID int64) ([]football.FixtureResponse, error)
}

func (m *MockFixtureFetcher) GetFixture(ctx context.Context, fixtureID int64) (*football.FixtureResponse, error) {
	if m.GetFixtureFunc != nil {
		return m.GetFixtureFunc(ctx, fixtureID)
	}
	panic("GetFixture called unexpectedly")
}

func (m *MockFixtureFetcher) GetFixturesByTeam(ctx context.Context, teamAPIID int64) ([]football.FixtureResponse, error) {
	if m.GetFixturesByTeamFunc != nil {
		return m.GetFixturesByTeamFunc(ctx, teamAPIID)
	}
	panic("GetFixturesByTeam called unexpectedly")
}
