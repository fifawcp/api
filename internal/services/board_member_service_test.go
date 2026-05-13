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

func newTestBoardMemberService(
	br *mocks.MockBoardRepository,
	bmr *mocks.MockBoardMemberRepository,
) BoardMemberServiceInterface {
	return NewBoardMemberService(br, bmr)
}

// ---------------------------------------------------------------------------
// TestBoardMemberService_JoinBoard
// ---------------------------------------------------------------------------
func TestBoardMemberService_JoinBoard(t *testing.T) {
	t.Parallel()

	t.Run("returns board ID on success", func(t *testing.T) {
		t.Parallel()

		joinCode := "ABCD1234"
		userID := gofakeit.UUID()
		expectedBoardID := gofakeit.Int64()

		bmr := &mocks.MockBoardMemberRepository{
			CreateBoardMemberFunc: func(ctx context.Context, jc string, uid string) (int64, error) {
				assert.Equal(t, joinCode, jc)
				assert.Equal(t, userID, uid)
				return expectedBoardID, nil
			},
		}

		service := newTestBoardMemberService(nil, bmr)

		boardID, err := service.JoinBoard(context.Background(), joinCode, userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedBoardID, boardID)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		joinCode := "ABCD1234"
		userID := gofakeit.UUID()

		bmr := &mocks.MockBoardMemberRepository{
			CreateBoardMemberFunc: func(ctx context.Context, jc string, uid string) (int64, error) {
				return 0, errors.New("database error")
			},
		}

		service := newTestBoardMemberService(nil, bmr)

		boardID, err := service.JoinBoard(context.Background(), joinCode, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Empty(t, boardID)
	})
}

// ---------------------------------------------------------------------------
// TestBoardMemberService_GetBoardMember
// ---------------------------------------------------------------------------
func TestBoardMemberService_GetBoardMember(t *testing.T) {
	t.Parallel()

	t.Run("returns board member on success", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()
		expectedBoard := &domain.Board{ID: boardID, Name: "Test Board"}
		expectedMember := &domain.BoardMember{
			BoardID: boardID,
			UserID:  userID,
			Role:    domain.BoardMemberRoleAdmin,
		}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				assert.Equal(t, boardID, bid)
				return expectedBoard, nil
			},
		}

		bmr := &mocks.MockBoardMemberRepository{
			GetBoardMemberFunc: func(ctx context.Context, bid int64, uid string) (*domain.BoardMember, error) {
				assert.Equal(t, boardID, bid)
				assert.Equal(t, userID, uid)
				return expectedMember, nil
			},
		}

		service := newTestBoardMemberService(br, bmr)

		result, err := service.GetBoardMember(context.Background(), boardID, userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedMember, result)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return nil, domain.ErrBoardNotFound
			},
		}

		service := newTestBoardMemberService(br, nil)

		result, err := service.GetBoardMember(context.Background(), boardID, userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrBoardNotFound)
	})

	t.Run("propagates board member repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: boardID}, nil
			},
		}

		bmr := &mocks.MockBoardMemberRepository{
			GetBoardMemberFunc: func(ctx context.Context, bid int64, uid string) (*domain.BoardMember, error) {
				return nil, errors.New("database error")
			},
		}

		service := newTestBoardMemberService(br, bmr)

		result, err := service.GetBoardMember(context.Background(), boardID, userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")
	})
}

// ---------------------------------------------------------------------------
// TestBoardMemberService_GetBoardMembers
// ---------------------------------------------------------------------------
func TestBoardMemberService_GetBoardMembers(t *testing.T) {
	t.Parallel()

	t.Run("forwards page/limit and returns members page", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		page := 2
		limit := 50

		expected := &domain.BoardMembersPage{
			Members: []*domain.BoardMemberDetails{
				{UserID: gofakeit.UUID()},
				{UserID: gofakeit.UUID()},
			},
			Pagination: domain.Pagination{Page: page, Limit: limit, Total: 2, HasMore: false},
		}

		filters := domain.BoardMembersFilters{Search: "alice"}
		br := &mocks.MockBoardRepository{
			GetBoardMembersFunc: func(ctx context.Context, bid int64, gotFilters domain.BoardMembersFilters, gotPage, gotLimit int) (*domain.BoardMembersPage, error) {
				assert.Equal(t, boardID, bid)
				assert.Equal(t, filters, gotFilters)
				assert.Equal(t, page, gotPage)
				assert.Equal(t, limit, gotLimit)
				return expected, nil
			},
		}

		service := newTestBoardMemberService(br, nil)

		result, err := service.GetBoardMembers(context.Background(), boardID, filters, page, limit)

		assert.NoError(t, err)
		assert.Same(t, expected, result)
	})

	t.Run("propagates board repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()

		br := &mocks.MockBoardRepository{
			GetBoardMembersFunc: func(ctx context.Context, bid int64, filters domain.BoardMembersFilters, page, limit int) (*domain.BoardMembersPage, error) {
				return nil, domain.ErrBoardNotFound
			},
		}

		service := newTestBoardMemberService(br, nil)

		result, err := service.GetBoardMembers(context.Background(), boardID, domain.BoardMembersFilters{}, 1, 20)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrBoardNotFound)
		assert.Nil(t, result)
	})
}

// ---------------------------------------------------------------------------
// TestBoardMemberService_UpdateBoardMemberRole
// ---------------------------------------------------------------------------
func TestBoardMemberService_UpdateBoardMemberRole(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on success for admin when updating member role", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()
		payload := dtos.UpdateBoardMemberRoleDto{Role: domain.BoardMemberRoleMember}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			UpdateBoardMemberRoleFunc: func(ctx context.Context, bid int64, uid string, role domain.BoardMemberRole) error {
				assert.Equal(t, boardID, bid)
				assert.Equal(t, userID, uid)
				assert.Equal(t, payload.Role, role)
				return nil
			},
		}

		service := newTestBoardMemberService(br, bmr)

		err := service.UpdateBoardMemberRole(context.Background(), boardID, userID, domain.BoardMemberRoleAdmin, payload)

		assert.NoError(t, err)
	})

	t.Run("returns forbidden for non-admin when updating member role", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()
		payload := dtos.UpdateBoardMemberRoleDto{Role: domain.BoardMemberRoleAdmin}

		service := newTestBoardMemberService(nil, nil)

		err := service.UpdateBoardMemberRole(context.Background(), boardID, userID, domain.BoardMemberRoleMember, payload)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrForbidden, err)
	})

	t.Run("rejects action when updating member role on global board", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()
		payload := dtos.UpdateBoardMemberRoleDto{Role: domain.BoardMemberRoleMember}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyGlobal}, nil
			},
		}

		service := newTestBoardMemberService(br, nil)

		err := service.UpdateBoardMemberRole(context.Background(), boardID, userID, domain.BoardMemberRoleAdmin, payload)

		assert.ErrorIs(t, err, domain.ErrBoardIsGlobal)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()
		payload := dtos.UpdateBoardMemberRoleDto{Role: domain.BoardMemberRoleMember}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			UpdateBoardMemberRoleFunc: func(ctx context.Context, bid int64, uid string, role domain.BoardMemberRole) error {
				return errors.New("database error")
			},
		}

		service := newTestBoardMemberService(br, bmr)

		err := service.UpdateBoardMemberRole(context.Background(), boardID, userID, domain.BoardMemberRoleAdmin, payload)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

// ---------------------------------------------------------------------------
// TestBoardMemberService_RemoveBoardMember
// ---------------------------------------------------------------------------
func TestBoardMemberService_RemoveBoardMember(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on success for admin", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			RemoveBoardMemberFunc: func(ctx context.Context, bid int64, uid string) error {
				assert.Equal(t, boardID, bid)
				assert.Equal(t, userID, uid)
				return nil
			},
		}

		service := newTestBoardMemberService(br, bmr)

		err := service.RemoveBoardMember(context.Background(), boardID, userID, domain.BoardMemberRoleAdmin)

		assert.NoError(t, err)
	})

	t.Run("returns forbidden for non-admin", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()

		service := newTestBoardMemberService(nil, nil)

		err := service.RemoveBoardMember(context.Background(), boardID, userID, domain.BoardMemberRoleMember)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrForbidden, err)
	})

	t.Run("rejects global board", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyGlobal}, nil
			},
		}

		service := newTestBoardMemberService(br, nil)

		err := service.RemoveBoardMember(context.Background(), boardID, userID, domain.BoardMemberRoleAdmin)

		assert.ErrorIs(t, err, domain.ErrBoardIsGlobal)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			RemoveBoardMemberFunc: func(ctx context.Context, bid int64, uid string) error {
				return errors.New("database error")
			},
		}

		service := newTestBoardMemberService(br, bmr)

		err := service.RemoveBoardMember(context.Background(), boardID, userID, domain.BoardMemberRoleAdmin)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

// ---------------------------------------------------------------------------
// TestBoardMemberService_LeaveBoard
// ---------------------------------------------------------------------------
func TestBoardMemberService_LeaveBoard(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on success", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			LeaveBoardFunc: func(ctx context.Context, bid int64, uid string) error {
				assert.Equal(t, boardID, bid)
				assert.Equal(t, userID, uid)
				return nil
			},
		}

		service := newTestBoardMemberService(br, bmr)

		err := service.LeaveBoard(context.Background(), boardID, userID)

		assert.NoError(t, err)
	})

	t.Run("rejects global board", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyGlobal}, nil
			},
		}

		service := newTestBoardMemberService(br, nil)

		err := service.LeaveBoard(context.Background(), boardID, userID)

		assert.ErrorIs(t, err, domain.ErrBoardIsGlobal)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			LeaveBoardFunc: func(ctx context.Context, bid int64, uid string) error {
				return errors.New("database error")
			},
		}

		service := newTestBoardMemberService(br, bmr)

		err := service.LeaveBoard(context.Background(), boardID, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

// ---------------------------------------------------------------------------
// TestBoardMemberService_TransferOwnership
// ---------------------------------------------------------------------------
func TestBoardMemberService_TransferOwnership(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on success for owner", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		callerUserID := gofakeit.UUID()
		targetUserID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			TransferOwnershipFunc: func(ctx context.Context, bid int64, oldOwnerID, newOwnerID string) error {
				assert.Equal(t, boardID, bid)
				assert.Equal(t, callerUserID, oldOwnerID)
				assert.Equal(t, targetUserID, newOwnerID)
				return nil
			},
		}

		service := newTestBoardMemberService(br, bmr)

		err := service.TransferOwnership(context.Background(), boardID, callerUserID, targetUserID, domain.BoardMemberRoleOwner)

		assert.NoError(t, err)
	})

	t.Run("returns forbidden for admin caller", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		callerUserID := gofakeit.UUID()
		targetUserID := gofakeit.UUID()

		service := newTestBoardMemberService(nil, nil)

		err := service.TransferOwnership(context.Background(), boardID, callerUserID, targetUserID, domain.BoardMemberRoleAdmin)

		assert.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("returns forbidden for member caller", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		callerUserID := gofakeit.UUID()
		targetUserID := gofakeit.UUID()

		service := newTestBoardMemberService(nil, nil)

		err := service.TransferOwnership(context.Background(), boardID, callerUserID, targetUserID, domain.BoardMemberRoleMember)

		assert.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("rejects transfer on global board", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		callerUserID := gofakeit.UUID()
		targetUserID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyGlobal}, nil
			},
		}

		service := newTestBoardMemberService(br, nil)

		err := service.TransferOwnership(context.Background(), boardID, callerUserID, targetUserID, domain.BoardMemberRoleOwner)

		assert.ErrorIs(t, err, domain.ErrBoardIsGlobal)
	})

	t.Run("rejects self-transfer", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}

		service := newTestBoardMemberService(br, nil)

		err := service.TransferOwnership(context.Background(), boardID, userID, userID, domain.BoardMemberRoleOwner)

		assert.ErrorIs(t, err, domain.ErrCannotTransferOwnershipToSelf)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.Int64()
		callerUserID := gofakeit.UUID()
		targetUserID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid int64) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			TransferOwnershipFunc: func(ctx context.Context, bid int64, oldOwnerID, newOwnerID string) error {
				return errors.New("database error")
			},
		}

		service := newTestBoardMemberService(br, bmr)

		err := service.TransferOwnership(context.Background(), boardID, callerUserID, targetUserID, domain.BoardMemberRoleOwner)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}
