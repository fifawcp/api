package domain

import "context"

// UserScore is the per-user aggregate of all score_events. One row per user.
// A row is created at user signup (in the same tx as the users INSERT) and
// updated by scoring runs. The leaderboard view per board is derived at read
// time by BoardRepository.GetBoardDetails (joins user_scores with board_members
// and applies RANK() partitioned by board).
type UserScore struct {
	UserID           string
	TotalPoints      int
	PickemPoints     int
	MatchScorePoints int
	ExactHits        int
	CorrectOutcomes  int
	UpdatedAt        string
}

type UserScoreRepository interface {
	BatchUpdateUserScores(ctx context.Context, userIDs []string, exactScorePts int) error
}
