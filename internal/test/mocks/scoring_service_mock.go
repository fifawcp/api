package mocks

import "context"

type MockScoringService struct {
	ScoreMatchesFunc    func(ctx context.Context, matchIDs []int64) error
	ScoreBestThirdsFunc func(ctx context.Context) error
}

func (m *MockScoringService) ScoreMatches(ctx context.Context, matchIDs []int64) error {
	if m.ScoreMatchesFunc != nil {
		return m.ScoreMatchesFunc(ctx, matchIDs)
	}
	panic("ScoreMatches called unexpectedly")
}

func (m *MockScoringService) ScoreBestThirds(ctx context.Context) error {
	if m.ScoreBestThirdsFunc != nil {
		return m.ScoreBestThirdsFunc(ctx)
	}
	panic("ScoreBestThirds called unexpectedly")
}
