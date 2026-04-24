package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

func newTestBoardRankingService(
	r *mocks.MockBoardRankingRepository,
) BoardRankingServiceInterface {
	return NewBoardRankingService(r)
}

// ---------------------------------------------------------------------------
// TestBoardRankingService_GetBoardRanking
// ---------------------------------------------------------------------------
func TestBoardRankingService_GetBoardRanking(t *testing.T) {
	t.Parallel()

	t.Run("returns board ranking on success", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		expectedRanking := []*domain.BoardRanking{
			{
				BoardID:         boardID,
				UserID:          gofakeit.UUID(),
				TotalPoints:     gofakeit.Number(1, 100),
				GlobalPoints:    gofakeit.Number(1, 100),
				DetailedPoints:  gofakeit.Number(1, 100),
				ExactHits:       gofakeit.Number(1, 100),
				CorrectOutcomes: gofakeit.Number(1, 100),
				UpdatedAt:       gofakeit.Date().Format(time.RFC3339),
			},
		}

		r := &mocks.MockBoardRankingRepository{
			GetBoardRankingFunc: func(ctx context.Context, id string) ([]*domain.BoardRanking, error) {
				assert.Equal(t, boardID, id)
				return expectedRanking, nil
			},
		}

		service := newTestBoardRankingService(r)

		result, err := service.GetBoardRanking(context.Background(), boardID)

		assert.NoError(t, err)
		assert.Equal(t, expectedRanking, result)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()

		r := &mocks.MockBoardRankingRepository{
			GetBoardRankingFunc: func(ctx context.Context, id string) ([]*domain.BoardRanking, error) {
				return nil, errors.New("database error")
			},
		}

		service := newTestBoardRankingService(r)

		result, err := service.GetBoardRanking(context.Background(), boardID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")
	})
}
