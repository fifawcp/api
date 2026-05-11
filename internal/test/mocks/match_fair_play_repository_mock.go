package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockMatchFairPlayRepository struct {
	UpsertFunc                   func(ctx context.Context, records []domain.MatchFairPlay) error
	GetFairPlayTotalsByGroupFunc  func(ctx context.Context, groupCode string) (map[string]int, error)
}

func (m *MockMatchFairPlayRepository) Upsert(ctx context.Context, records []domain.MatchFairPlay) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, records)
	}
	panic("Upsert called unexpectedly")
}

func (m *MockMatchFairPlayRepository) GetFairPlayTotalsByGroup(ctx context.Context, groupCode string) (map[string]int, error) {
	if m.GetFairPlayTotalsByGroupFunc != nil {
		return m.GetFairPlayTotalsByGroupFunc(ctx, groupCode)
	}
	panic("GetFairPlayTotalsByGroup called unexpectedly")
}
