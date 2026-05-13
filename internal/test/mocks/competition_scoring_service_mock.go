package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockCompetitionScoringService struct {
	RecomputeForMatchesFunc    func(ctx context.Context, result *domain.ScoreMatchesResult) error
	RecomputeForBestThirdsFunc func(ctx context.Context, affectedUserIDs []string) error
}

func (m *MockCompetitionScoringService) RecomputeForMatches(ctx context.Context, result *domain.ScoreMatchesResult) error {
	if m.RecomputeForMatchesFunc != nil {
		return m.RecomputeForMatchesFunc(ctx, result)
	}
	panic("RecomputeForMatches called unexpectedly")
}

func (m *MockCompetitionScoringService) RecomputeForBestThirds(ctx context.Context, affectedUserIDs []string) error {
	if m.RecomputeForBestThirdsFunc != nil {
		return m.RecomputeForBestThirdsFunc(ctx, affectedUserIDs)
	}
	panic("RecomputeForBestThirds called unexpectedly")
}
