package domain

import (
	"context"
	"time"
)

type Board struct {
	ID          string    `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string    `json:"name" example:"My Board"`
	OwnerUserID string    `json:"owner_user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	JoinCode    string    `json:"join_code" example:"ABCD1234"`
	CreatedAt   time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}

type BoardRepository interface {
	CreateBoardWithOwner(ctx context.Context, board *Board) error
	GetUserBoards(ctx context.Context, userID string) ([]*Board, error)
	GetBoardByID(ctx context.Context, boardID string) (*Board, error)
	UpdateJoinCode(ctx context.Context, boardID string, joinCode string) error
	UpdateBoard(ctx context.Context, boardID string, board *Board) error
	DeleteBoard(ctx context.Context, boardID string, userID string) error
}
