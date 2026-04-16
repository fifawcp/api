package domain

import "context"

type BoardRanking struct {
	BoardID         string
	UserID          string
	TotalPoints     int
	GlobalPoints    int
	DetailedPoints  int
	ExactHits       int
	CorrectOutcomes int
	UpdatedAt       string
}

type BoardRankingRepository interface {
	GetBoardRanking(ctx context.Context, boardID string) ([]*BoardRanking, error)
}
