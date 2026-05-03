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

type BoardDetails struct {
	ID          string                `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string                `json:"name" example:"My Board"`
	OwnerUserID string                `json:"owner_user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	JoinCode    string                `json:"join_code" example:"ABCD1234"`
	Privacy     string                `json:"privacy" example:"private"`
	CreatedAt   time.Time             `json:"created_at" example:"2026-01-15T10:30:00Z"`
	JoinedAt    time.Time             `json:"joined_at" example:"2026-01-16T10:30:00Z"`
	UserRank    int                   `json:"user_rank" example:"3"`
	Members     []*BoardDetailsMember `json:"members"`
}

type BoardDetailsMember struct {
	UserID           string          `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	UserName         string          `json:"username" example:"johndoe"`
	FirstName        string          `json:"first_name" example:"John"`
	LastName         string          `json:"last_name" example:"Doe"`
	Role             BoardMemberRole `json:"role" example:"member"`
	JoinedAt         time.Time       `json:"joined_at" example:"2026-01-16T10:30:00Z"`
	Rank             int             `json:"rank" example:"3"`
	TotalPoints      int             `json:"total_points" example:"100"`
	PickemPoints     int             `json:"pickem_points" example:"50"`
	MatchScorePoints int             `json:"match_score_points" example:"50"`
	ExactHits        int             `json:"exact_hits" example:"5"`
	CorrectOutcomes  int             `json:"correct_outcomes" example:"10"`
	UpdatedAt        time.Time       `json:"updated_at" example:"2026-01-20T10:30:00Z"`
}

type BoardRepository interface {
	CreateBoardWithOwner(ctx context.Context, board *Board) error
	GetUserBoards(ctx context.Context, userID string) ([]*Board, error)
	GetBoardByID(ctx context.Context, boardID string) (*Board, error)
	GetBoardDetails(ctx context.Context, boardID string) (*BoardDetails, error)
	UpdateJoinCode(ctx context.Context, boardID string, joinCode string) error
	UpdateBoard(ctx context.Context, boardID string, board *Board) error
	DeleteBoard(ctx context.Context, boardID string, userID string) error
}
