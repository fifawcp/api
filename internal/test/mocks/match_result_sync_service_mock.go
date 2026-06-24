package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/football"
)

type MockMatchResultSyncService struct {
	SyncMatchFunc        func(ctx context.Context, matchID int64) (*domain.MatchSyncResult, error)
	ResolveFixtureIDFunc func(ctx context.Context, match *domain.Match) (int64, error)
	FinalizeFunc         func(ctx context.Context, matchID, fixtureID int64, fixture *football.FixtureResponse) (*domain.SyncGroupStageOutcomes, error)
}

func (m *MockMatchResultSyncService) SyncMatch(ctx context.Context, matchID int64) (*domain.MatchSyncResult, error) {
	if m.SyncMatchFunc != nil {
		return m.SyncMatchFunc(ctx, matchID)
	}
	panic("SyncMatch called unexpectedly")
}

func (m *MockMatchResultSyncService) ResolveFixtureID(ctx context.Context, match *domain.Match) (int64, error) {
	if m.ResolveFixtureIDFunc != nil {
		return m.ResolveFixtureIDFunc(ctx, match)
	}
	panic("ResolveFixtureID called unexpectedly")
}

func (m *MockMatchResultSyncService) Finalize(ctx context.Context, matchID, fixtureID int64, fixture *football.FixtureResponse) (*domain.SyncGroupStageOutcomes, error) {
	if m.FinalizeFunc != nil {
		return m.FinalizeFunc(ctx, matchID, fixtureID, fixture)
	}
	panic("Finalize called unexpectedly")
}
