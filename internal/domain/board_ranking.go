package domain

import "context"

type BoardRanking struct {
	BoardID         string `json:"board_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	UserID          string `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	TotalPoints     int    `json:"total_points" example:"100"`
	GlobalPoints    int    `json:"global_points" example:"50"`
	DetailedPoints  int    `json:"detailed_points" example:"50"`
	ExactHits       int    `json:"exact_hits" example:"5"`
	CorrectOutcomes int    `json:"correct_outcomes" example:"10"`
	UpdatedAt       string `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type BoardRankingRepository interface {
	GetBoardRanking(ctx context.Context, boardID string) ([]*BoardRanking, error)
}
