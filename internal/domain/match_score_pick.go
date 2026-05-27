package domain

import "context"

type UserMatchScorePick struct {
	UserID    string `json:"user_id"`
	MatchID   int64  `json:"match_id"`
	HomeScore int    `json:"home_score"`
	AwayScore int    `json:"away_score"`
}

type MatchScorePickRepository interface {
	UpsertMatchScorePick(ctx context.Context, pick *UserMatchScorePick) error
	GetMatchScorePicksByUser(ctx context.Context, userID string) ([]*UserMatchScorePick, error)
	GetMatchScorePicksByMatch(ctx context.Context, matchID int64) ([]*UserMatchScorePick, error)
	CountMatchScorePicksByUser(ctx context.Context, userID string) (int, error)
}
