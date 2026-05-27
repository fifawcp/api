package domain

import (
	"context"
	"time"
)

type BoardPrivacy string

const (
	BoardPrivacyPublic  BoardPrivacy = "public"
	BoardPrivacyPrivate BoardPrivacy = "private"
	BoardPrivacyGlobal  BoardPrivacy = "global"
)

type Board struct {
	ID        int64        `json:"id" example:"1"`
	Name      string       `json:"name" example:"My Board"`
	JoinCode  *string      `json:"join_code,omitempty" example:"ABCD1234"`
	Privacy   BoardPrivacy `json:"privacy" example:"private"`
	CreatedAt time.Time    `json:"created_at" example:"2026-01-15T10:30:00Z"`
}

type UserBoardListItem struct {
	ID      int64        `json:"id" example:"1"`
	Name    string       `json:"name" example:"My Board"`
	Privacy BoardPrivacy `json:"privacy" example:"private"`
}

type BoardDetails struct {
	Board
	MemberCount      int         `json:"member_count" example:"12"`
	CompetitionCount int         `json:"competition_count" example:"2"`
	Viewer           BoardViewer `json:"viewer"`
}

type BoardViewer struct {
	Role     BoardMemberRole `json:"role" example:"member"`
	JoinedAt time.Time       `json:"joined_at" example:"2026-01-16T10:30:00Z"`
}

type BoardMembersPage struct {
	Members    []*BoardMemberDetails
	Pagination Pagination
}

type BoardMembersFilters struct {
	Search string
}

type BoardMemberDetails struct {
	UserID    string          `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	UserName  string          `json:"username" example:"johndoe"`
	FirstName string          `json:"first_name" example:"John"`
	LastName  string          `json:"last_name" example:"Doe"`
	Role      BoardMemberRole `json:"role" example:"member"`
	JoinedAt  time.Time       `json:"joined_at" example:"2026-01-16T10:30:00Z"`
}

type BoardRepository interface {
	CreateBoard(ctx context.Context, board *Board, ownerID string) error
	GetUserBoards(ctx context.Context, userID string) ([]*UserBoardListItem, error)
	GetBoardByID(ctx context.Context, boardID int64) (*Board, error)
	GetBoardDetails(ctx context.Context, boardID int64, userID string) (*BoardDetails, error)
	GetBoardMembers(ctx context.Context, boardID int64, filters BoardMembersFilters, page, limit int) (*BoardMembersPage, error)
	UpdateJoinCode(ctx context.Context, boardID int64, joinCode string) error
	UpdateBoard(ctx context.Context, boardID int64, board *Board) error
	DeleteBoard(ctx context.Context, boardID int64) error
}
