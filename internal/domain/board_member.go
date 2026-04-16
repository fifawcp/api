package domain

import (
	"context"
	"time"
)

type BoardMember struct {
	BoardID   string
	UserID    string
	Role      BoardMemberRole
	CreatedAt time.Time
	UserName  string // optional, populated when joining user
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
