package services

import (
	"context"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

func newTestPickemService(repo *mocks.MockPickemRepository, lockTime time.Time) PickemServiceInterface {
	cfg := &config.Config{}
	logger := &mocks.MockLogger{}
	return NewPickemService(repo, nil, lockTime, cfg, logger)
}

// repoWithEmptyPicks returns a MockPickemRepository whose four parallel-fetch
// methods all return empty slices — sufficient to let GetUserPickem succeed.
func repoWithEmptyPicks() *mocks.MockPickemRepository {
	return &mocks.MockPickemRepository{
		GetGroupPicksFunc: func(ctx context.Context, userID string) ([]*domain.UserGroupPick, error) {
			return nil, nil
		},
		GetBestThirdPicksFunc: func(ctx context.Context, userID string) ([]*domain.UserBestThirdPick, error) {
			return nil, nil
		},
		GetBracketPicksFunc: func(ctx context.Context, userID string) ([]*domain.UserBracketPick, error) {
			return nil, nil
		},
		GetLockedGroupCodesFunc: func(ctx context.Context, userID string) ([]string, error) {
			return nil, nil
		},
	}
}

// ---------------------------------------------------------------------------
// GetMemberPickem
// ---------------------------------------------------------------------------
func TestPickemService_GetMemberPickem_HiddenBeforeLock(t *testing.T) {
	t.Parallel()

	service := newTestPickemService(&mocks.MockPickemRepository{}, time.Now().UTC().Add(24*time.Hour))

	result, err := service.GetMemberPickem(context.Background(), gofakeit.UUID())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrPredictionsHidden)
}

func TestPickemService_GetMemberPickem_ReturnsPickemAfterLock(t *testing.T) {
	t.Parallel()

	service := newTestPickemService(repoWithEmptyPicks(), time.Now().UTC().Add(-24*time.Hour))

	result, err := service.GetMemberPickem(context.Background(), gofakeit.UUID())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsLocked)
}
