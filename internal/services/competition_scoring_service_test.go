package services

import (
	"context"
	"testing"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompetitionScoringService_RecomputeForMatches_Picks(t *testing.T) {
	t.Parallel()

	matchIDs := []int64{42}
	userIDs := []string{"user-1", "user-2"}
	pickID := int64(7)

	var upsertedPickID int64
	var upsertedPts int
	var upsertedUsers []string

	scoreRepo := &mocks.MockCompetitionScoreRepository{
		FindMatchCompetitionsByMatchesFunc: func(ctx context.Context, ids []int64) ([]int64, error) {
			return nil, nil
		},
		FindPickCompetitionsByMatchesFunc: func(ctx context.Context, ids []int64) ([]int64, error) {
			assert.Equal(t, matchIDs, ids)
			return []int64{pickID}, nil
		},
		BatchUpsertPickScoresFunc: func(ctx context.Context, competitionID int64, users []string, exactScorePts int) error {
			upsertedPickID = competitionID
			upsertedUsers = users
			upsertedPts = exactScorePts
			return nil
		},
	}

	cfg := &config.Config{Scoring: config.ScoringConfig{MatchScoreExact: 5}}
	service := NewCompetitionScoringService(scoreRepo, cfg, &mocks.MockLogger{})

	err := service.RecomputeForMatches(context.Background(), &domain.ScoreMatchesResult{
		AffectedUserIDs: userIDs,
		ScoredMatchIDs:  matchIDs,
	})

	require.NoError(t, err)
	assert.Equal(t, pickID, upsertedPickID)
	assert.Equal(t, userIDs, upsertedUsers)
	assert.Equal(t, 5, upsertedPts)
}
