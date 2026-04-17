package mocks

import (
	"context"

	"github.com/ncondes/fifawcp/internal/domain"
	"github.com/ncondes/fifawcp/internal/dtos"
)

type MockBoardMemberService struct {
	JoinBoardFunc             func(ctx context.Context, joinCode string, userID string) error
	GetBoardMemberFunc        func(ctx context.Context, boardID string, userID string) (*domain.BoardMember, error)
	GetBoardMembersFunc       func(ctx context.Context, boardID string) ([]*domain.BoardMember, error)
	UpdateBoardMemberRoleFunc func(ctx context.Context, boardID string, userID string, role domain.BoardMemberRole, payload dtos.UpdateBoardMemberRoleDto) error
	RemoveBoardMemberFunc     func(ctx context.Context, boardID string, userID string, role domain.BoardMemberRole) error
}

func (m *MockBoardMemberService) JoinBoard(ctx context.Context, joinCode string, userID string) error {
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

func (m *MockBoardMemberService) GetBoardMembers(ctx context.Context, boardID string) ([]*domain.BoardMember, error) {
	if m.GetBoardMembersFunc != nil {
		return m.GetBoardMembersFunc(ctx, boardID)
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
