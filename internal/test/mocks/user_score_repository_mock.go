package mocks

import (
	"context"
)

type MockUserScoreRepository struct {
	BatchUpdateUserScoresFunc func(ctx context.Context, userIDs []string, exactScorePts int) error
}

func (m *MockUserScoreRepository) BatchUpdateUserScores(ctx context.Context, userIDs []string, exactScorePts int) error {
	if m.BatchUpdateUserScoresFunc != nil {
		return m.BatchUpdateUserScoresFunc(ctx, userIDs, exactScorePts)
	}
	panic("BatchUpdateUserScores called unexpectedly")
}
