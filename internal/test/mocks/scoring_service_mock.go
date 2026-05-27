package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockScoringService struct {
	ScoreMatchesFunc    func(ctx context.Context, matchIDs []int64) (*domain.ScoreMatchesResult, error)
	ScoreBestThirdsFunc func(ctx context.Context) ([]string, error)
}

func (m *MockScoringService) ScoreMatches(ctx context.Context, matchIDs []int64) (*domain.ScoreMatchesResult, error) {
	if m.ScoreMatchesFunc != nil {
		return m.ScoreMatchesFunc(ctx, matchIDs)
	}
	panic("ScoreMatches called unexpectedly")
}

func (m *MockScoringService) ScoreBestThirds(ctx context.Context) ([]string, error) {
	if m.ScoreBestThirdsFunc != nil {
		return m.ScoreBestThirdsFunc(ctx)
	}
	panic("ScoreBestThirds called unexpectedly")
}
