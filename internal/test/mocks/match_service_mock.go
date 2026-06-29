package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

type MockMatchService struct {
	GetMatchesFunc                 func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error)
	UpdateMatchResultFunc          func(ctx context.Context, matchID int64, payload dtos.UpdateMatchResultDto) (*domain.SyncGroupStageOutcomes, error)
	UpdateMatchResultsBulkFunc     func(ctx context.Context, payload dtos.BulkUpdateMatchesResultDto) (*domain.SyncGroupStageOutcomes, error)
	UpdateMatchResultsBulkSyncFunc func(ctx context.Context, payload dtos.BulkUpdateMatchesResultDto) (*domain.SyncGroupStageOutcomes, error)
	ResetMatchResultFunc           func(ctx context.Context, matchID int64) (*domain.SyncGroupStageOutcomes, error)
	SyncGroupStageOutcomesFunc     func(ctx context.Context) (*domain.SyncGroupStageOutcomes, error)
	ResolveThirdPlaceConflictFunc  func(ctx context.Context, payload dtos.ResolveThirdPlaceConflictDto) (*domain.SyncGroupStageOutcomes, error)
	AdvanceBracketFunc             func(ctx context.Context, completedMatchIDs []int64) error
}

func (m *MockMatchService) GetMatches(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
	if m.GetMatchesFunc != nil {
		return m.GetMatchesFunc(ctx, filters)
	}
	panic("GetMatches called unexpectedly")
}

func (m *MockMatchService) UpdateMatchResult(ctx context.Context, matchID int64, payload dtos.UpdateMatchResultDto) (*domain.SyncGroupStageOutcomes, error) {
	if m.UpdateMatchResultFunc != nil {
		return m.UpdateMatchResultFunc(ctx, matchID, payload)
	}
	panic("UpdateMatchResult called unexpectedly")
}

func (m *MockMatchService) UpdateMatchResultsBulk(ctx context.Context, payload dtos.BulkUpdateMatchesResultDto) (*domain.SyncGroupStageOutcomes, error) {
	if m.UpdateMatchResultsBulkFunc != nil {
		return m.UpdateMatchResultsBulkFunc(ctx, payload)
	}
	panic("UpdateMatchResultsBulk called unexpectedly")
}

func (m *MockMatchService) UpdateMatchResultsBulkSync(ctx context.Context, payload dtos.BulkUpdateMatchesResultDto) (*domain.SyncGroupStageOutcomes, error) {
	if m.UpdateMatchResultsBulkSyncFunc != nil {
		return m.UpdateMatchResultsBulkSyncFunc(ctx, payload)
	}
	panic("UpdateMatchResultsBulkSync called unexpectedly")
}

func (m *MockMatchService) ResetMatchResult(ctx context.Context, matchID int64) (*domain.SyncGroupStageOutcomes, error) {
	if m.ResetMatchResultFunc != nil {
		return m.ResetMatchResultFunc(ctx, matchID)
	}
	panic("ResetMatchResult called unexpectedly")
}

func (m *MockMatchService) SyncGroupStageOutcomes(ctx context.Context) (*domain.SyncGroupStageOutcomes, error) {
	if m.SyncGroupStageOutcomesFunc != nil {
		return m.SyncGroupStageOutcomesFunc(ctx)
	}
	panic("SyncGroupStageOutcomes called unexpectedly")
}

func (m *MockMatchService) ResolveThirdPlaceConflict(ctx context.Context, payload dtos.ResolveThirdPlaceConflictDto) (*domain.SyncGroupStageOutcomes, error) {
	if m.ResolveThirdPlaceConflictFunc != nil {
		return m.ResolveThirdPlaceConflictFunc(ctx, payload)
	}
	panic("ResolveThirdPlaceConflict called unexpectedly")
}

func (m *MockMatchService) AdvanceBracket(ctx context.Context, completedMatchIDs []int64) error {
	if m.AdvanceBracketFunc != nil {
		return m.AdvanceBracketFunc(ctx, completedMatchIDs)
	}
	panic("AdvanceBracket called unexpectedly")
}
