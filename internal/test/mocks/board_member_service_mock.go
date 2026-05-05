package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

type MockBoardMemberService struct {
	JoinBoardFunc             func(ctx context.Context, joinCode string, userID string) (string, error)
	GetBoardMemberFunc        func(ctx context.Context, boardID string, userID string) (*domain.BoardMember, error)
	GetBoardMembersFunc       func(ctx context.Context, boardID string, page, limit int) (*domain.BoardMembersPage, error)
	UpdateBoardMemberRoleFunc func(ctx context.Context, boardID string, userID string, role domain.BoardMemberRole, payload dtos.UpdateBoardMemberRoleDto) error
	RemoveBoardMemberFunc     func(ctx context.Context, boardID string, userID string, role domain.BoardMemberRole) error
	LeaveBoardFunc            func(ctx context.Context, boardID string, userID string) error
}

func (m *MockBoardMemberService) JoinBoard(ctx context.Context, joinCode string, userID string) (string, error) {
	if m.JoinBoardFunc != nil {
		return m.JoinBoardFunc(ctx, joinCode, userID)
	}
	panic("JoinBoard called unexpectedly")
}

func (m *MockBoardMemberService) GetBoardMember(ctx context.Context, boardID string, userID string) (*domain.BoardMember, error) {
	if m.GetBoardMemberFunc != nil {
		return m.GetBoardMemberFunc(ctx, boardID, userID)
	}
	panic("GetBoardMember called unexpectedly")
}

func (m *MockBoardMemberService) GetBoardMembers(ctx context.Context, boardID string, page, limit int) (*domain.BoardMembersPage, error) {
	if m.GetBoardMembersFunc != nil {
		return m.GetBoardMembersFunc(ctx, boardID, page, limit)
	}
	panic("GetBoardMembers called unexpectedly")
}

func (m *MockBoardMemberService) UpdateBoardMemberRole(ctx context.Context, boardID string, userID string, role domain.BoardMemberRole, payload dtos.UpdateBoardMemberRoleDto) error {
	if m.UpdateBoardMemberRoleFunc != nil {
		return m.UpdateBoardMemberRoleFunc(ctx, boardID, userID, role, payload)
	}
	panic("UpdateBoardMemberRole called unexpectedly")
}

func (m *MockBoardMemberService) RemoveBoardMember(ctx context.Context, boardID string, userID string, role domain.BoardMemberRole) error {
	if m.RemoveBoardMemberFunc != nil {
		return m.RemoveBoardMemberFunc(ctx, boardID, userID, role)
	}
	panic("RemoveBoardMember called unexpectedly")
}

func (m *MockBoardMemberService) LeaveBoard(ctx context.Context, boardID string, userID string) error {
	if m.LeaveBoardFunc != nil {
		return m.LeaveBoardFunc(ctx, boardID, userID)
	}
	panic("LeaveBoard called unexpectedly")
}
