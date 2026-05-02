package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

type MockBoardService struct {
	CreateBoardFunc        func(ctx context.Context, payload dtos.CreateBoardDto, userID string) (*domain.Board, error)
	GetUserBoardsFunc      func(ctx context.Context, userID string) ([]*domain.BoardSummary, error)
	GetBoardByIDFunc       func(ctx context.Context, boardID string) (*domain.BoardDetails, error)
	RegenerateJoinCodeFunc func(ctx context.Context, boardID string) (string, error)
	UpdateBoardFunc        func(ctx context.Context, boardID string, role domain.BoardMemberRole, payload dtos.UpdateBoardDto) error
	DeleteBoardFunc        func(ctx context.Context, boardID string, userID string) error
}

func (m *MockBoardService) CreateBoard(ctx context.Context, payload dtos.CreateBoardDto, userID string) (*domain.Board, error) {
	if m.CreateBoardFunc != nil {
		return m.CreateBoardFunc(ctx, payload, userID)
	}
	panic("CreateBoard called unexpectedly")
}

func (m *MockBoardService) GetUserBoards(ctx context.Context, userID string) ([]*domain.BoardSummary, error) {
	if m.GetUserBoardsFunc != nil {
		return m.GetUserBoardsFunc(ctx, userID)
	}
	panic("GetUserBoards called unexpectedly")
}

func (m *MockBoardService) GetBoardByID(ctx context.Context, boardID string) (*domain.BoardDetails, error) {
	if m.GetBoardByIDFunc != nil {
		return m.GetBoardByIDFunc(ctx, boardID)
	}
	panic("GetBoardByID called unexpectedly")
}

func (m *MockBoardService) RegenerateJoinCode(ctx context.Context, boardID string) (string, error) {
	if m.RegenerateJoinCodeFunc != nil {
		return m.RegenerateJoinCodeFunc(ctx, boardID)
	}
	panic("RegenerateJoinCode called unexpectedly")
}

func (m *MockBoardService) UpdateBoard(ctx context.Context, boardID string, role domain.BoardMemberRole, payload dtos.UpdateBoardDto) error {
	if m.UpdateBoardFunc != nil {
		return m.UpdateBoardFunc(ctx, boardID, role, payload)
	}
	panic("UpdateBoard called unexpectedly")
}

func (m *MockBoardService) DeleteBoard(ctx context.Context, boardID string, userID string) error {
	if m.DeleteBoardFunc != nil {
		return m.DeleteBoardFunc(ctx, boardID, userID)
	}
	panic("DeleteBoard called unexpectedly")
}
