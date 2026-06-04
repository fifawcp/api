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

func TestCompetitionScoringService_RecomputeForMatches_Pools(t *testing.T) {
	t.Parallel()

	matchIDs := []int64{42}
	userIDs := []string{"user-1", "user-2"}
	poolID := int64(7)

	var upsertedPoolID int64
	var upsertedPts int
	var upsertedUsers []string

	scoreRepo := &mocks.MockCompetitionScoreRepository{
		FindMatchCompetitionsByMatchesFunc: func(ctx context.Context, ids []int64) ([]int64, error) {
			return nil, nil
		},
		FindPoolCompetitionsByMatchesFunc: func(ctx context.Context, ids []int64) ([]int64, error) {
			assert.Equal(t, matchIDs, ids)
			return []int64{poolID}, nil
		},
		BatchUpsertPoolScoresFunc: func(ctx context.Context, competitionID int64, users []string, exactScorePts int) error {
			upsertedPoolID = competitionID
			upsertedUsers = users
			upsertedPts = exactScorePts
			return nil
		},
	}

	cfg := &config.Config{Scoring: config.ScoringConfig{MatchScoreExact: 5}}
	service := NewCompetitionScoringService(&mocks.MockCompetitionRepository{}, scoreRepo, cfg, &mocks.MockLogger{})

	err := service.RecomputeForMatches(context.Background(), &domain.ScoreMatchesResult{
		AffectedUserIDs: userIDs,
		ScoredMatchIDs:  matchIDs,
		PickemAffected:  false,
	})

	require.NoError(t, err)
	assert.Equal(t, poolID, upsertedPoolID)
	assert.Equal(t, userIDs, upsertedUsers)
	assert.Equal(t, 5, upsertedPts)
}
