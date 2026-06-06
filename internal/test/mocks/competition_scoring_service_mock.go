package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockCompetitionScoringService struct {
	RecomputeForMatchesFunc func(ctx context.Context, result *domain.ScoreMatchesResult) error
}

func (m *MockCompetitionScoringService) RecomputeForMatches(ctx context.Context, result *domain.ScoreMatchesResult) error {
	if m.RecomputeForMatchesFunc != nil {
		return m.RecomputeForMatchesFunc(ctx, result)
	}
	panic("RecomputeForMatches called unexpectedly")
}
