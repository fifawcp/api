package domain

import "context"

// ScoreMatchesResult is the output of a scoring run, used by competition scoring
// to drive downstream recomputes without re-querying the database
type ScoreMatchesResult struct {
	AffectedUserIDs []string
	ScoredMatchIDs  []int64
	PickemAffected  bool
}

type CompetitionScoreRepository interface {
	FindMatchCompetitionsByMatches(ctx context.Context, matchIDs []int64) ([]int64, error)
	BatchUpsertMatchScores(ctx context.Context, competitionID int64, userIDs []string, exactScorePts int) error
	BatchUpsertPickemScores(ctx context.Context, competitionIDs []int64, userIDs []string) error
	GetLeaderboard(ctx context.Context, competitionID int64, page, limit int) (*CompetitionLeaderboardPage, error)
}
