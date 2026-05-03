package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockBoardRepository struct {
	CreateBoardWithOwnerFunc func(ctx context.Context, board *domain.Board) error
	GetUserBoardsFunc        func(ctx context.Context, userID string) ([]*domain.UserBoardListItem, error)
	GetBoardByIDFunc         func(ctx context.Context, boardID string) (*domain.Board, error)
	GetBoardDetailsFunc      func(ctx context.Context, boardID string, userID string) (*domain.BoardDetails, error)
	GetBoardMembersFunc      func(ctx context.Context, boardID string, page, limit int) (*domain.BoardMembersPage, error)
	UpdateJoinCodeFunc       func(ctx context.Context, boardID string, joinCode string) error
	UpdateBoardFunc          func(ctx context.Context, boardID string, board *domain.Board) error
	DeleteBoardFunc          func(ctx context.Context, boardID string, userID string) error
}

func (m *MockBoardRepository) CreateBoardWithOwner(ctx context.Context, board *domain.Board) error {
	if m.CreateBoardWithOwnerFunc != nil {
		return m.CreateBoardWithOwnerFunc(ctx, board)
	}
	panic("CreateBoardWithOwner called unexpectedly")
}

func (m *MockBoardRepository) GetUserBoards(ctx context.Context, userID string) ([]*domain.UserBoardListItem, error) {
	if m.GetUserBoardsFunc != nil {
		return m.GetUserBoardsFunc(ctx, userID)
	}
	panic("GetUserBoards called unexpectedly")
}

func (m *MockBoardRepository) GetBoardByID(ctx context.Context, boardID string) (*domain.Board, error) {
	if m.GetBoardByIDFunc != nil {
		return m.GetBoardByIDFunc(ctx, boardID)
	}
	panic("GetBoardByID called unexpectedly")
}

func (m *MockBoardRepository) GetBoardDetails(ctx context.Context, boardID string, userID string) (*domain.BoardDetails, error) {
	if m.GetBoardDetailsFunc != nil {
		return m.GetBoardDetailsFunc(ctx, boardID, userID)
	}
	panic("GetBoardDetails called unexpectedly")
}

func (m *MockBoardRepository) GetBoardMembers(ctx context.Context, boardID string, page, limit int) (*domain.BoardMembersPage, error) {
	if m.GetBoardMembersFunc != nil {
		return m.GetBoardMembersFunc(ctx, boardID, page, limit)
	}
	panic("GetBoardMembers called unexpectedly")
}

func (m *MockBoardRepository) UpdateJoinCode(ctx context.Context, boardID string, joinCode string) error {
	if m.UpdateJoinCodeFunc != nil {
		return m.UpdateJoinCodeFunc(ctx, boardID, joinCode)
	}
	panic("UpdateJoinCode called unexpectedly")
}

func (m *MockBoardRepository) UpdateBoard(ctx context.Context, boardID string, board *domain.Board) error {
	if m.UpdateBoardFunc != nil {
		return m.UpdateBoardFunc(ctx, boardID, board)
	}
	panic("UpdateBoard called unexpectedly")
}

func (m *MockBoardRepository) DeleteBoard(ctx context.Context, boardID string, userID string) error {
	if m.DeleteBoardFunc != nil {
		return m.DeleteBoardFunc(ctx, boardID, userID)
	}
	panic("DeleteBoard called unexpectedly")
}
