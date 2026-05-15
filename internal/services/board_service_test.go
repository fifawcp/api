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
		expectedID := gofakeit.Int64()

		br := &mocks.MockBoardRepository{
			CreateBoardFunc: func(ctx context.Context, board *domain.Board, ownerUserID string) error {
				board.ID = expectedID
				return nil
			},
		}

		service := newTestBoardService(br)

		result, err := service.CreateBoard(context.Background(), payload, userID)

		assert.NoError(t, err)
		assert.Equal(t, payload.Name, result.Name)
		assert.NotNil(t, result.JoinCode)
		assert.Len(t, *result.JoinCode, 8)
		assert.Equal(t, domain.BoardPrivacyPrivate, result.Privacy)
	})

	t.Run("retries on join code collision and returns error after max retries", func(t *testing.T) {
		t.Parallel()

		payload := dtos.CreateBoardDto{Name: "Test Board"}
		userID := gofakeit.UUID()

		attemptCount := 0
		br := &mocks.MockBoardRepository{
			CreateBoardFunc: func(ctx context.Context, board *domain.Board, ownerUserID string) error {
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
			CreateBoardFunc: func(ctx context.Context, board *domain.Board, ownerUserID string) error {
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
		expectedBoards := []*domain.UserBoardListItem{
			{ID: gofakeit.Int64(), Name: "Board 1"},
			{ID: gofakeit.Int64(), Name: "Board 2"},
		}

		br := &mocks.MockBoardRepository{
			GetUserBoardsFunc: func(ctx context.Context, uid string) ([]*domain.UserBoardListItem, error) {
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
			GetUserBoardsFunc: func(ctx context.Context, uid string) ([]*domain.UserBoardListItem, error) {
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

	t.Run("returns board metadata + requesting user's rank/joined_at on success", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()
		joinedAt := gofakeit.Date()

		repoBoard := &domain.BoardDetails{
			Board: domain.Board{
				ID:      boardID,
				Name:    "Test Board",
				Privacy: domain.BoardPrivacyPrivate,
			},
			Viewer: domain.BoardViewer{
				Role:     domain.BoardMemberRoleAdmin,
				JoinedAt: joinedAt,
			},
		}

		br := &mocks.MockBoardRepository{
			GetBoardDetailsFunc: func(ctx context.Context, bid int64, uid string) (*domain.BoardDetails, error) {
				assert.Equal(t, boardID, bid)
				assert.Equal(t, userID, uid)
				return repoBoard, nil
			},
		}

		service := newTestBoardService(br)

		result, err := service.GetBoardByID(context.Background(), boardID, userID)

		assert.NoError(t, err)
		assert.Equal(t, domain.BoardPrivacyPrivate, result.Privacy)
		assert.Equal(t, domain.BoardMemberRoleAdmin, result.Viewer.Role)
		assert.Equal(t, joinedAt, result.Viewer.JoinedAt)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardDetailsFunc: func(ctx context.Context, bid int64, uid string) (*domain.BoardDetails, error) {
				return nil, domain.ErrBoardNotFound
			},
		}

		service := newTestBoardService(br)

		result, err := service.GetBoardByID(context.Background(), boardID, userID)

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

		boardID := gofakeit.Int64()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
			UpdateJoinCodeFunc: func(ctx context.Context, bid int64, joinCode string) error {
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

	t.Run("rejects global board", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyGlobal}, nil
			},
		}

		service := newTestBoardService(br)

		result, err := service.RegenerateJoinCode(context.Background(), boardID)

		assert.ErrorIs(t, err, domain.ErrBoardIsGlobal)
		assert.Empty(t, result)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
			UpdateJoinCodeFunc: func(ctx context.Context, bid int64, joinCode string) error {
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

		boardID := gofakeit.Int64()
		payload := dtos.UpdateBoardDto{Name: "Updated Board"}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
			UpdateBoardFunc: func(ctx context.Context, bid int64, board *domain.Board) error {
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

		boardID := gofakeit.Int64()
		payload := dtos.UpdateBoardDto{Name: "Updated Board"}

		service := newTestBoardService(nil)

		err := service.UpdateBoard(context.Background(), boardID, domain.BoardMemberRoleMember, payload)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrForbidden, err)
	})

	t.Run("rejects global board", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		payload := dtos.UpdateBoardDto{Name: "Updated Board"}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyGlobal}, nil
			},
		}

		service := newTestBoardService(br)

		err := service.UpdateBoard(context.Background(), boardID, domain.BoardMemberRoleAdmin, payload)

		assert.ErrorIs(t, err, domain.ErrBoardIsGlobal)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		payload := dtos.UpdateBoardDto{Name: "Updated Board"}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
			UpdateBoardFunc: func(ctx context.Context, bid int64, board *domain.Board) error {
				return errors.New("database error")
			},
		}

		service := newTestBoardService(br)

		err := service.UpdateBoard(context.Background(), boardID, domain.BoardMemberRoleAdmin, payload)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})

	t.Run("propagates assert private board error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		payload := dtos.UpdateBoardDto{Name: "Updated Board"}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return nil, errors.New("database error")
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

		boardID := gofakeit.Int64()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
			DeleteBoardFunc: func(ctx context.Context, bid int64) error {
				assert.Equal(t, boardID, bid)
				return nil
			},
		}

		service := newTestBoardService(br)

		err := service.DeleteBoard(context.Background(), boardID, domain.BoardMemberRoleOwner)

		assert.NoError(t, err)
	})

	t.Run("rejects global board", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyGlobal}, nil
			},
		}

		service := newTestBoardService(br)

		err := service.DeleteBoard(context.Background(), boardID, domain.BoardMemberRoleOwner)

		assert.ErrorIs(t, err, domain.ErrBoardIsGlobal)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
			DeleteBoardFunc: func(ctx context.Context, bid int64) error {
				return errors.New("database error")
			},
		}

		service := newTestBoardService(br)

		err := service.DeleteBoard(context.Background(), boardID, domain.BoardMemberRoleOwner)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
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
