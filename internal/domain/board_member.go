package domain

import (
	"context"
	"time"
)

type BoardMember struct {
	BoardID   string          `json:"board_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	UserID    string          `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Role      BoardMemberRole `json:"role" example:"member"`
	CreatedAt time.Time       `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UserName  string          `json:"username" example:"johndoe"` // optional, populated when joining user
}

type BoardMemberRole string

const (
	BoardMemberRoleAdmin  BoardMemberRole = "admin"
	BoardMemberRoleMember BoardMemberRole = "member"
)

type BoardMemberRepository interface {
	CreateBoardMember(ctx context.Context, joinCode string, userID string) error
	GetBoardMember(ctx context.Context, boardID string, userID string) (*BoardMember, error)
	GetBoardMembers(ctx context.Context, boardID string) ([]*BoardMember, error)
	UpdateBoardMemberRole(ctx context.Context, boardID string, userID string, role BoardMemberRole) error
	RemoveBoardMember(ctx context.Context, boardID string, userID string) error
}
