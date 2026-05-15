package mocks

import (
	"context"
	"time"

	"github.com/fifawcp/api/internal/domain"
)

type MockMatchRepository struct {
	GetMatchesFunc                     func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error)
	GetFirstGroupStageMatchKickoffFunc func(ctx context.Context) (time.Time, error)
	GetNextScheduledMatchFunc          func(ctx context.Context) (*domain.Match, error)
	UpdateMatchesResultFunc            func(ctx context.Context, updates []domain.MatchResultUpdate) error
	UpdateMatchTeamsFunc               func(ctx context.Context, updates []domain.MatchTeamUpdate) error
	ResetMatchResultFunc               func(ctx context.Context, matchID int64) error
	IsGroupFinishedFunc                func(ctx context.Context, groupCode string) (bool, error)
	IsGroupStageFinishedFunc           func(ctx context.Context) (bool, error)
}

func (m *MockMatchRepository) GetMatches(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
	if m.GetMatchesFunc != nil {
		return m.GetMatchesFunc(ctx, filters)
	}
	panic("GetMatches called unexpectedly")
}

func (m *MockMatchRepository) GetFirstGroupStageMatchKickoff(ctx context.Context) (time.Time, error) {
	if m.GetFirstGroupStageMatchKickoffFunc != nil {
		return m.GetFirstGroupStageMatchKickoffFunc(ctx)
	}
	panic("GetFirstGroupStageMatchKickoff called unexpectedly")
}

func (m *MockMatchRepository) GetNextScheduledMatch(ctx context.Context) (*domain.Match, error) {
	if m.GetNextScheduledMatchFunc != nil {
		return m.GetNextScheduledMatchFunc(ctx)
	}
	panic("GetNextScheduledMatch called unexpectedly")
}

func (m *MockMatchRepository) UpdateMatchTeams(ctx context.Context, updates []domain.MatchTeamUpdate) error {
	if m.UpdateMatchTeamsFunc != nil {
		return m.UpdateMatchTeamsFunc(ctx, updates)
	}
	panic("UpdateMatchTeams called unexpectedly")
}

func (m *MockMatchRepository) UpdateMatchesResult(ctx context.Context, updates []domain.MatchResultUpdate) error {
	if m.UpdateMatchesResultFunc != nil {
		return m.UpdateMatchesResultFunc(ctx, updates)
	}
	panic("UpdateMatchesResult called unexpectedly")
}

func (m *MockMatchRepository) ResetMatchResult(ctx context.Context, matchID int64) error {
	if m.ResetMatchResultFunc != nil {
		return m.ResetMatchResultFunc(ctx, matchID)
	}
	panic("ResetMatchResult called unexpectedly")
}

func (m *MockMatchRepository) IsGroupFinished(ctx context.Context, groupCode string) (bool, error) {
	if m.IsGroupFinishedFunc != nil {
		return m.IsGroupFinishedFunc(ctx, groupCode)
	}
	panic("IsGroupFinished called unexpectedly")
}

func (m *MockMatchRepository) IsGroupStageFinished(ctx context.Context) (bool, error) {
	if m.IsGroupStageFinishedFunc != nil {
		return m.IsGroupStageFinishedFunc(ctx)
	}
	panic("IsGroupStageFinished called unexpectedly")
}
