package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockBoardMemberRepository struct {
	CreateBoardMemberFunc     func(ctx context.Context, joinCode string, userID string) error
	GetBoardMemberFunc        func(ctx context.Context, boardID string, userID string) (*domain.BoardMember, error)
	GetBoardMembersFunc       func(ctx context.Context, boardID string) ([]*domain.BoardMember, error)
	UpdateBoardMemberRoleFunc func(ctx context.Context, boardID string, userID string, role domain.BoardMemberRole) error
	RemoveBoardMemberFunc     func(ctx context.Context, boardID string, userID string) error
}

func (m *MockBoardMemberRepository) CreateBoardMember(ctx context.Context, joinCode string, userID string) error {
	if m.CreateBoardMemberFunc != nil {
		return m.CreateBoardMemberFunc(ctx, joinCode, userID)
	}
	panic("CreateBoardMember called unexpectedly")
}

func (m *MockBoardMemberRepository) GetBoardMember(ctx context.Context, boardID string, userID string) (*domain.BoardMember, error) {
	if m.GetBoardMemberFunc != nil {
		return m.GetBoardMemberFunc(ctx, boardID, userID)
	}
	panic("GetBoardMember called unexpectedly")
}

func (m *MockBoardMemberRepository) GetBoardMembers(ctx context.Context, boardID string) ([]*domain.BoardMember, error) {
	if m.GetBoardMembersFunc != nil {
		return m.GetBoardMembersFunc(ctx, boardID)
	}
	panic("GetBoardMembers called unexpectedly")
}

func (m *MockBoardMemberRepository) UpdateBoardMemberRole(ctx context.Context, boardID string, userID string, role domain.BoardMemberRole) error {
	if m.UpdateBoardMemberRoleFunc != nil {
		return m.UpdateBoardMemberRoleFunc(ctx, boardID, userID, role)
	}
	panic("UpdateBoardMemberRole called unexpectedly")
}

func (m *MockBoardMemberRepository) RemoveBoardMember(ctx context.Context, boardID string, userID string) error {
	if m.RemoveBoardMemberFunc != nil {
		return m.RemoveBoardMemberFunc(ctx, boardID, userID)
	}
	panic("RemoveBoardMember called unexpectedly")
}
