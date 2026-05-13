package mocks

import (
	"context"
)

type MockMatchAPIFixtureRepository struct {
	GetByMatchIDFunc    func(ctx context.Context, matchID int64) (int64, error)
	UpsertFixtureIDFunc func(ctx context.Context, matchID, apiFixtureID int64) error
}

func (m *MockMatchAPIFixtureRepository) GetByMatchID(ctx context.Context, matchID int64) (int64, error) {
	if m.GetByMatchIDFunc != nil {
		return m.GetByMatchIDFunc(ctx, matchID)
	}
	panic("GetByMatchID called unexpectedly")
}

func (m *MockMatchAPIFixtureRepository) UpsertFixtureID(ctx context.Context, matchID, apiFixtureID int64) error {
	if m.UpsertFixtureIDFunc != nil {
		return m.UpsertFixtureIDFunc(ctx, matchID, apiFixtureID)
	}
	panic("UpsertFixtureID called unexpectedly")
}
