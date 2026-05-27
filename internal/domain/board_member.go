package domain

import (
	"context"
	"time"
)

type BoardMember struct {
	BoardID   int64           `json:"board_id" example:"1"`
	UserID    string          `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Role      BoardMemberRole `json:"role" example:"member"`
	CreatedAt time.Time       `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UserName  string          `json:"username" example:"johndoe"` // optional, populated when joining user
}

type BoardMemberRole string

const (
	BoardMemberRoleOwner  BoardMemberRole = "owner"
	BoardMemberRoleAdmin  BoardMemberRole = "admin"
	BoardMemberRoleMember BoardMemberRole = "member"
)

func (r BoardMemberRole) CanManage() bool {
	return r == BoardMemberRoleOwner || r == BoardMemberRoleAdmin
}

type BoardMemberRepository interface {
	CreateBoardMember(ctx context.Context, joinCode string, userID string) (int64, error)
	GetBoardMember(ctx context.Context, boardID int64, userID string) (*BoardMember, error)
	UpdateBoardMemberRole(ctx context.Context, boardID int64, userID string, role BoardMemberRole) error
	RemoveBoardMember(ctx context.Context, boardID int64, userID string) error
	LeaveBoard(ctx context.Context, boardID int64, userID string) error
	TransferOwnership(ctx context.Context, boardID int64, oldOwnerUserID, newOwnerUserID string) error
}
