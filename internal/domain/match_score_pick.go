package domain

import "context"

type UserMatchScorePick struct {
	UserID    string `json:"user_id"`
	MatchID   int64  `json:"match_id"`
	HomeScore int    `json:"home_score"`
	AwayScore int    `json:"away_score"`
}

// BoardMemberMatchPick pairs a board member with their (possibly absent) score pick for a match.
// HomeScore and AwayScore are nil when the member has not submitted a pick.
type BoardMemberMatchPick struct {
	Member    CompetitionLeaderboardMember
	HomeScore *int
	AwayScore *int
}

type MatchScorePickRepository interface {
	UpsertMatchScorePick(ctx context.Context, pick *UserMatchScorePick) error
	GetMatchScorePicksByUser(ctx context.Context, userID string) ([]*UserMatchScorePick, error)
	GetMatchScorePicksByUserAndMatches(ctx context.Context, userID string, matchIDs []int64) ([]*UserMatchScorePick, error)
	GetMatchScorePicksByMatch(ctx context.Context, matchID int64) ([]*UserMatchScorePick, error)
	CountMatchScorePicksByUser(ctx context.Context, userID string) (int, error)
	GetBoardMembersMatchPicks(ctx context.Context, boardID, matchID int64) ([]*BoardMemberMatchPick, error)
}
