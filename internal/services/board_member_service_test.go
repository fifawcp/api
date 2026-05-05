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
		expectedBoardID := gofakeit.UUID()

		bmr := &mocks.MockBoardMemberRepository{
			CreateBoardMemberFunc: func(ctx context.Context, jc string, uid string) (string, error) {
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
			CreateBoardMemberFunc: func(ctx context.Context, jc string, uid string) (string, error) {
				return "", errors.New("database error")
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

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()
		expectedBoard := &domain.Board{ID: boardID, Name: "Test Board"}
		expectedMember := &domain.BoardMember{
			BoardID: boardID,
			UserID:  userID,
			Role:    domain.BoardMemberRoleAdmin,
		}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				assert.Equal(t, boardID, bid)
				return expectedBoard, nil
			},
		}

		bmr := &mocks.MockBoardMemberRepository{
			GetBoardMemberFunc: func(ctx context.Context, bid string, uid string) (*domain.BoardMember, error) {
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

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
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

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				return &domain.Board{ID: boardID}, nil
			},
		}

		bmr := &mocks.MockBoardMemberRepository{
			GetBoardMemberFunc: func(ctx context.Context, bid string, uid string) (*domain.BoardMember, error) {
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

		boardID := gofakeit.UUID()
		page := 2
		limit := 50

		expected := &domain.BoardMembersPage{
			Members: []*domain.BoardMemberDetails{
				{UserID: gofakeit.UUID(), Rank: 1},
				{UserID: gofakeit.UUID(), Rank: 2},
			},
			Pagination: domain.Pagination{Page: page, Limit: limit, Total: 2, HasMore: false},
		}

		filters := domain.BoardMembersFilters{Search: "alice", Sort: domain.BoardMembersSortMatchScorePoints}
		br := &mocks.MockBoardRepository{
			GetBoardMembersFunc: func(ctx context.Context, bid string, gotFilters domain.BoardMembersFilters, gotPage, gotLimit int) (*domain.BoardMembersPage, error) {
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

		boardID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardMembersFunc: func(ctx context.Context, bid string, filters domain.BoardMembersFilters, page, limit int) (*domain.BoardMembersPage, error) {
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

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()
		payload := dtos.UpdateBoardMemberRoleDto{Role: domain.BoardMemberRoleMember}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			UpdateBoardMemberRoleFunc: func(ctx context.Context, bid string, uid string, role domain.BoardMemberRole) error {
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

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()
		payload := dtos.UpdateBoardMemberRoleDto{Role: domain.BoardMemberRoleAdmin}

		service := newTestBoardMemberService(nil, nil)

		err := service.UpdateBoardMemberRole(context.Background(), boardID, userID, domain.BoardMemberRoleMember, payload)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrForbidden, err)
	})

	t.Run("rejects action when updating member role on public board", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()
		payload := dtos.UpdateBoardMemberRoleDto{Role: domain.BoardMemberRoleMember}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPublic}, nil
			},
		}

		service := newTestBoardMemberService(br, nil)

		err := service.UpdateBoardMemberRole(context.Background(), boardID, userID, domain.BoardMemberRoleAdmin, payload)

		assert.ErrorIs(t, err, domain.ErrBoardIsPublic)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()
		payload := dtos.UpdateBoardMemberRoleDto{Role: domain.BoardMemberRoleMember}

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			UpdateBoardMemberRoleFunc: func(ctx context.Context, bid string, uid string, role domain.BoardMemberRole) error {
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

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			RemoveBoardMemberFunc: func(ctx context.Context, bid string, uid string) error {
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

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()

		service := newTestBoardMemberService(nil, nil)

		err := service.RemoveBoardMember(context.Background(), boardID, userID, domain.BoardMemberRoleMember)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrForbidden, err)
	})

	t.Run("rejects public board", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPublic}, nil
			},
		}

		service := newTestBoardMemberService(br, nil)

		err := service.RemoveBoardMember(context.Background(), boardID, userID, domain.BoardMemberRoleAdmin)

		assert.ErrorIs(t, err, domain.ErrBoardIsPublic)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			RemoveBoardMemberFunc: func(ctx context.Context, bid string, uid string) error {
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

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			LeaveBoardFunc: func(ctx context.Context, bid string, uid string) error {
				assert.Equal(t, boardID, bid)
				assert.Equal(t, userID, uid)
				return nil
			},
		}

		service := newTestBoardMemberService(br, bmr)

		err := service.LeaveBoard(context.Background(), boardID, userID)

		assert.NoError(t, err)
	})

	t.Run("rejects public board", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPublic}, nil
			},
		}

		service := newTestBoardMemberService(br, nil)

		err := service.LeaveBoard(context.Background(), boardID, userID)

		assert.ErrorIs(t, err, domain.ErrBoardIsPublic)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()

		br := &mocks.MockBoardRepository{
			GetBoardByIDFunc: func(ctx context.Context, bid string) (*domain.Board, error) {
				return &domain.Board{ID: bid, Privacy: domain.BoardPrivacyPrivate}, nil
			},
		}
		bmr := &mocks.MockBoardMemberRepository{
			LeaveBoardFunc: func(ctx context.Context, bid string, uid string) error {
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
// TestBoardMemberService_isAdminMember
// ---------------------------------------------------------------------------
func TestBoardMemberService_isAdminMember(t *testing.T) {
	t.Parallel()

	service := NewBoardMemberService(nil, nil).(*BoardMemberService)

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
