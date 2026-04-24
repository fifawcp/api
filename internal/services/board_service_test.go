package services

import (
	"context"
	"errors"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

func newTestBoardService(br *mocks.MockBoardRepository) BoardServiceInterface {
	return NewBoardService(br)
}

// ---------------------------------------------------------------------------
// TestBoardService_CreateBoard
// ---------------------------------------------------------------------------
func TestBoardService_CreateBoard(t *testing.T) {
	t.Parallel()

	t.Run("returns board on success", func(t *testing.T) {
		t.Parallel()

		payload := dtos.CreateBoardDto{Name: "Test Board"}
		userID := gofakeit.UUID()
		expectedBoard := &domain.Board{
			ID:          gofakeit.UUID(),
			Name:        payload.Name,
			OwnerUserID: userID,
			JoinCode:    "ABCD1234",
		}

		br := &mocks.MockBoardRepository{
			CreateBoardWithOwnerFunc: func(ctx context.Context, board *domain.Board) error {
				board.ID = expectedBoard.ID
				return nil
			},
		}

		service := newTestBoardService(br)

		result, err := service.CreateBoard(context.Background(), payload, userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedBoard.Name, result.Name)
		assert.Equal(t, expectedBoard.OwnerUserID, result.OwnerUserID)
		assert.NotEmpty(t, result.JoinCode)
		assert.Len(t, result.JoinCode, 8)
	})

	t.Run("retries on join code collision and returns error after max retries", func(t *testing.T) {
		t.Parallel()

		payload := dtos.CreateBoardDto{Name: "Test Board"}
		userID := gofakeit.UUID()

		attemptCount := 0
		br := &mocks.MockBoardRepository{
			CreateBoardWithOwnerFunc: func(ctx context.Context, board *domain.Board) error {
				attemptCount++
				return domain.ErrBoardAlreadyExists
			},
		}

		service := newTestBoardService(br)

		result, err := service.CreateBoard(context.Background(), payload, userID)

		assert.Error(t, err)
		assert.Equal(t, 5, attemptCount)
		assert.Nil(t, result)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		payload := dtos.CreateBoardDto{Name: "Test Board"}
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			CreateBoardWithOwnerFunc: func(ctx context.Context, board *domain.Board) error {
				return errors.New("database error")
			},
		}

		service := newTestBoardService(br)

		result, err := service.CreateBoard(context.Background(), payload, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, result)
	})
}

// ---------------------------------------------------------------------------
// TestBoardService_GetUserBoards
// ---------------------------------------------------------------------------
func TestBoardService_GetUserBoards(t *testing.T) {
	t.Parallel()

	t.Run("returns boards on success", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		expectedBoards := []*domain.Board{
			{ID: gofakeit.UUID(), Name: "Board 1", OwnerUserID: userID},
			{ID: gofakeit.UUID(), Name: "Board 2", OwnerUserID: userID},
		}

		br := &mocks.MockBoardRepository{
			GetUserBoardsFunc: func(ctx context.Context, uid string) ([]*domain.Board, error) {
				assert.Equal(t, userID, uid)
				return expectedBoards, nil
			},
		}

		service := newTestBoardService(br)

		result, err := service.GetUserBoards(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedBoards, result)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetUserBoardsFunc: func(ctx context.Context, uid string) ([]*domain.Board, error) {
				return nil, errors.New("database error")
			},
		}

		service := newTestBoardService(br)

		result, err := service.GetUserBoards(context.Background(), userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, result)
	})
}

// ---------------------------------------------------------------------------
// TestBoardService_GetBoardByID
// ---------------------------------------------------------------------------
func TestBoardService_GetBoardByID(t *testing.T) {
	t.Parallel()

	t.Run("returns board on success", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		expectedBoard := &domain.Board{
			ID:   boardID,
			Name: "Test Board",
		}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				assert.Equal(t, boardID, bid)
				return expectedBoard, nil
			},
		}

		service := newTestBoardService(br)

		result, err := service.GetBoardByID(context.Background(), boardID)

		assert.NoError(t, err)
		assert.Equal(t, expectedBoard, result)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				return nil, domain.ErrBoardNotFound
			},
		}

		service := newTestBoardService(br)

		result, err := service.GetBoardByID(context.Background(), boardID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrBoardNotFound)
		assert.Nil(t, result)
	})
}

// ---------------------------------------------------------------------------
// TestBoardService_RegenerateJoinCode
// ---------------------------------------------------------------------------
func TestBoardService_RegenerateJoinCode(t *testing.T) {
	t.Parallel()

	t.Run("returns new join code on success", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			UpdateJoinCodeFunc: func(ctx context.Context, bid string, joinCode string) error {
				assert.Equal(t, boardID, bid)
				assert.NotEmpty(t, joinCode)
				assert.Len(t, joinCode, 8)
				return nil
			},
		}

		service := newTestBoardService(br)

		result, err := service.RegenerateJoinCode(context.Background(), boardID)

		assert.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.Len(t, result, 8)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			UpdateJoinCodeFunc: func(ctx context.Context, bid string, joinCode string) error {
				return errors.New("database error")
			},
		}

		service := newTestBoardService(br)

		result, err := service.RegenerateJoinCode(context.Background(), boardID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Empty(t, result)
	})
}

// ---------------------------------------------------------------------------
// TestBoardService_UpdateBoard
// ---------------------------------------------------------------------------
func TestBoardService_UpdateBoard(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on success for admin", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		payload := dtos.UpdateBoardDto{Name: "Updated Board"}

		br := &mocks.MockBoardRepository{
			UpdateBoardFunc: func(ctx context.Context, bid string, board *domain.Board) error {
				assert.Equal(t, boardID, bid)
				assert.Equal(t, payload.Name, board.Name)
				return nil
			},
		}

		service := newTestBoardService(br)

		err := service.UpdateBoard(context.Background(), boardID, domain.BoardMemberRoleAdmin, payload)

		assert.NoError(t, err)
	})

	t.Run("returns forbidden for non-admin", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		payload := dtos.UpdateBoardDto{Name: "Updated Board"}

		service := newTestBoardService(nil)

		err := service.UpdateBoard(context.Background(), boardID, domain.BoardMemberRoleMember, payload)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrForbidden, err)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		payload := dtos.UpdateBoardDto{Name: "Updated Board"}

		br := &mocks.MockBoardRepository{
			UpdateBoardFunc: func(ctx context.Context, bid string, board *domain.Board) error {
				return errors.New("database error")
			},
		}

		service := newTestBoardService(br)

		err := service.UpdateBoard(context.Background(), boardID, domain.BoardMemberRoleAdmin, payload)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

// ---------------------------------------------------------------------------
// TestBoardService_DeleteBoard
// ---------------------------------------------------------------------------
func TestBoardService_DeleteBoard(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on success", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			DeleteBoardFunc: func(ctx context.Context, bid string, uid string) error {
				assert.Equal(t, boardID, bid)
				assert.Equal(t, userID, uid)
				return nil
			},
		}

		service := newTestBoardService(br)

		err := service.DeleteBoard(context.Background(), boardID, userID)

		assert.NoError(t, err)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			DeleteBoardFunc: func(ctx context.Context, bid string, uid string) error {
				return errors.New("database error")
			},
		}

		service := newTestBoardService(br)

		err := service.DeleteBoard(context.Background(), boardID, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

// ---------------------------------------------------------------------------
// TestBoardService_isAdminMember
// ---------------------------------------------------------------------------
func TestBoardService_isAdminMember(t *testing.T) {
	t.Parallel()

	service := NewBoardService(nil).(*BoardService)

	t.Run("returns true for admin role", func(t *testing.T) {
		t.Parallel()

		result := service.isAdminMember(domain.BoardMemberRoleAdmin)

		assert.True(t, result)
	})

	t.Run("returns false for member role", func(t *testing.T) {
		t.Parallel()

		result := service.isAdminMember(domain.BoardMemberRoleMember)

		assert.False(t, result)
	})
}

// ---------------------------------------------------------------------------
// TestBoardService_generateJoinCode
// ---------------------------------------------------------------------------
func TestBoardService_generateJoinCode(t *testing.T) {
	t.Parallel()

	service := NewBoardService(nil).(*BoardService)

	t.Run("generates 8 character code", func(t *testing.T) {
		t.Parallel()

		result := service.generateJoinCode()

		assert.Len(t, result, 8)
		assert.Regexp(t, "^[0-9A-Z]{8}$", result)
	})

	t.Run("generates unique codes", func(t *testing.T) {
		t.Parallel()

		codes := make(map[string]bool)
		for i := 0; i < 100; i++ {
			result := service.generateJoinCode()
			assert.False(t, codes[result], "code should be unique")
			codes[result] = true
		}
	})
}
