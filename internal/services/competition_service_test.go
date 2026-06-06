package services

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCompetitionService(br *mocks.MockBoardRepository, cr *mocks.MockCompetitionRepository) CompetitionServiceInterface {
	return NewCompetitionService(br, cr, nil)
}

func privateBoardRepo() *mocks.MockBoardRepository {
	return &mocks.MockBoardRepository{
		GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
			return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
		},
	}
}

// ---------------------------------------------------------------------------
// TestCompetitionService_CreateCompetition
// ---------------------------------------------------------------------------
func TestCompetitionService_CreateCompetition(t *testing.T) {
	t.Parallel()

	t.Run("creates a pick competition carrying the match id", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()
		matchID := gofakeit.Int64()

		var captured *domain.Competition
		cr := &mocks.MockCompetitionRepository{
			CreateCompetitionFunc: func(ctx context.Context, c *domain.Competition) error {
				captured = c
				return nil
			},
		}

		service := newTestCompetitionService(privateBoardRepo(), cr)

		_, err := service.CreateCompetition(context.Background(), boardID, userID, domain.BoardMemberRoleAdmin, dtos.CreateCompetitionDto{
			Type:    domain.CompetitionTypePick,
			Name:    "CvP",
			MatchID: &matchID,
		})

		require.NoError(t, err)
		require.NotNil(t, captured)
		require.NotNil(t, captured.PickMatchID)
		assert.Equal(t, matchID, *captured.PickMatchID)
		assert.Equal(t, domain.CompetitionTypePick, captured.Type)
	})

	t.Run("rejects a member without manage permission", func(t *testing.T) {
		t.Parallel()

		// Empty repos: the role check must short-circuit before any repo call.
		service := newTestCompetitionService(&mocks.MockBoardRepository{}, &mocks.MockCompetitionRepository{})

		_, err := service.CreateCompetition(context.Background(), gofakeit.Int64(), gofakeit.UUID(), domain.BoardMemberRoleMember, dtos.CreateCompetitionDto{
			Type: domain.CompetitionTypePick,
			Name: "CvP",
		})

		assert.ErrorIs(t, err, domain.ErrCompetitionForbidden)
	})
}

// ---------------------------------------------------------------------------
// TestCompetitionService_DeleteCompetition
// ---------------------------------------------------------------------------
func TestCompetitionService_DeleteCompetition(t *testing.T) {
	t.Parallel()

	t.Run("deletes the tournament pick'em competition", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		competitionID := gofakeit.Int64()
		deleteCalled := false

		cr := &mocks.MockCompetitionRepository{
			GetCompetitionByIDFunc: func(ctx context.Context, bid, cid int64) (*domain.Competition, error) {
				return &domain.Competition{ID: cid, BoardID: bid, Type: domain.CompetitionTypePickem}, nil
			},
			DeleteCompetitionFunc: func(ctx context.Context, bid, cid int64) error {
				deleteCalled = true
				return nil
			},
		}

		service := newTestCompetitionService(privateBoardRepo(), cr)

		err := service.DeleteCompetition(context.Background(), boardID, competitionID, domain.BoardMemberRoleAdmin)

		assert.NoError(t, err)
		assert.True(t, deleteCalled)
	})

	t.Run("deletes a match competition", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		competitionID := gofakeit.Int64()
		deleteCalled := false

		cr := &mocks.MockCompetitionRepository{
			GetCompetitionByIDFunc: func(ctx context.Context, bid, cid int64) (*domain.Competition, error) {
				return &domain.Competition{ID: cid, BoardID: bid, Type: domain.CompetitionTypeMatch}, nil
			},
			DeleteCompetitionFunc: func(ctx context.Context, bid, cid int64) error {
				deleteCalled = true
				assert.Equal(t, boardID, bid)
				assert.Equal(t, competitionID, cid)
				return nil
			},
		}

		service := newTestCompetitionService(privateBoardRepo(), cr)

		err := service.DeleteCompetition(context.Background(), boardID, competitionID, domain.BoardMemberRoleAdmin)

		assert.NoError(t, err)
		assert.True(t, deleteCalled)
	})

	t.Run("rejects a member without manage permission", func(t *testing.T) {
		t.Parallel()

		// Empty repos: the role check must short-circuit before any repo call.
		service := newTestCompetitionService(&mocks.MockBoardRepository{}, &mocks.MockCompetitionRepository{})

		err := service.DeleteCompetition(context.Background(), gofakeit.Int64(), gofakeit.Int64(), domain.BoardMemberRoleMember)

		assert.ErrorIs(t, err, domain.ErrCompetitionForbidden)
	})
}
