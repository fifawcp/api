package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockBoardMemberRepository struct {
	CreateBoardMemberFunc     func(ctx context.Context, joinCode string, userID string) (int64, error)
	GetBoardMemberFunc        func(ctx context.Context, boardID int64, userID string) (*domain.BoardMember, error)
	UpdateBoardMemberRoleFunc func(ctx context.Context, boardID int64, userID string, role domain.BoardMemberRole) error
	RemoveBoardMemberFunc     func(ctx context.Context, boardID int64, userID string) error
	LeaveBoardFunc            func(ctx context.Context, boardID int64, userID string) error
	TransferOwnershipFunc     func(ctx context.Context, boardID int64, oldOwnerUserID, newOwnerUserID string) error
}

func (m *MockBoardMemberRepository) CreateBoardMember(ctx context.Context, joinCode string, userID string) (int64, error) {
	if m.CreateBoardMemberFunc != nil {
		return m.CreateBoardMemberFunc(ctx, joinCode, userID)
	}
	panic("CreateBoardMember called unexpectedly")
}

func (m *MockBoardMemberRepository) GetBoardMember(ctx context.Context, boardID int64, userID string) (*domain.BoardMember, error) {
	if m.GetBoardMemberFunc != nil {
		return m.GetBoardMemberFunc(ctx, boardID, userID)
	}
	panic("GetBoardMember called unexpectedly")
}

func (m *MockBoardMemberRepository) UpdateBoardMemberRole(ctx context.Context, boardID int64, userID string, role domain.BoardMemberRole) error {
	if m.UpdateBoardMemberRoleFunc != nil {
		return m.UpdateBoardMemberRoleFunc(ctx, boardID, userID, role)
	}
	panic("UpdateBoardMemberRole called unexpectedly")
}

func (m *MockBoardMemberRepository) RemoveBoardMember(ctx context.Context, boardID int64, userID string) error {
	if m.RemoveBoardMemberFunc != nil {
		return m.RemoveBoardMemberFunc(ctx, boardID, userID)
	}
	panic("RemoveBoardMember called unexpectedly")
}

func (m *MockBoardMemberRepository) LeaveBoard(ctx context.Context, boardID int64, userID string) error {
	if m.LeaveBoardFunc != nil {
		return m.LeaveBoardFunc(ctx, boardID, userID)
	}
	panic("LeaveBoard called unexpectedly")
}

func (m *MockBoardMemberRepository) TransferOwnership(ctx context.Context, boardID int64, oldOwnerUserID, newOwnerUserID string) error {
	if m.TransferOwnershipFunc != nil {
		return m.TransferOwnershipFunc(ctx, boardID, oldOwnerUserID, newOwnerUserID)
	}
	panic("TransferOwnership called unexpectedly")
}
