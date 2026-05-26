package domain

import "context"

type UserPickem struct {
	GroupPicks []ResolvedGroupPick `json:"group_picks"`
	BestThirds []*Team             `json:"best_thirds"`
	Bracket    []*BracketMatchSlot `json:"bracket"`
	Progress   PickemProgress      `json:"progress"`
	IsLocked   bool                `json:"is_locked"`
}

type ResolvedGroupPick struct {
	GroupCode string       `json:"group_code"`
	Locked    bool         `json:"locked"`
	Teams     []RankedTeam `json:"teams"`
}

type RankedTeam struct {
	Team
	Position int `json:"position"`
}

type PickemProgress struct {
	Groups     StepProgress `json:"groups"`
	BestThirds StepProgress `json:"best_thirds"`
	Bracket    StepProgress `json:"bracket"`
}

type PickemProgressCounts struct {
	Groups     int
	BestThirds int
	Bracket    int
}

type StepProgress struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
}

func (s StepProgress) IsComplete() bool {
	return s.Completed == s.Total
}

type UserGroupPick struct {
	UserID            string `json:"user_id"`
	TeamFifaCode      string `json:"team_fifa_code"`
	TeamGroupCode     string `json:"team_group_code"`
	PredictedPosition int    `json:"predicted_position"`
}

type UserBestThirdPick struct {
	UserID       string `json:"user_id"`
	TeamFifaCode string `json:"team_fifa_code"`
}

type UserBracketPick struct {
	UserID       string `json:"user_id"`
	MatchID      int64  `json:"match_id"`
	TeamFifaCode string `json:"team_fifa_code"`
}

// Bracket view (computed at read time, never persisted)
type BracketMatchSlot struct {
	MatchID    int64          `json:"match_id"`
	StageCode  MatchStageCode `json:"stage_code"`
	HomeTeam   *Team          `json:"home_team"`   // projected from picks; nil if not yet resolvable
	AwayTeam   *Team          `json:"away_team"`   // projected from picks; nil if not yet resolvable
	PickedTeam *Team          `json:"picked_team"` // user's chosen winner; nil if not yet picked
}

type PickemRepository interface {
	UpsertGroupPicks(ctx context.Context, userID string, picks []*UserGroupPick) error
	GetGroupPicks(ctx context.Context, userID string) ([]*UserGroupPick, error)
	GetGroupPicksByGroup(ctx context.Context, groupCode string) ([]*UserGroupPick, error) // for scoring
	UpsertBestThirds(ctx context.Context, userID string, bestThirds []*UserBestThirdPick) error
	GetBestThirdPicks(ctx context.Context, userID string) ([]*UserBestThirdPick, error)
	GetBestThirdPicksByTeams(ctx context.Context, teamFifaCodes []string) ([]*UserBestThirdPick, error) // for ScoreBestThirds
	UpsertBracketPicks(ctx context.Context, userID string, picks []*UserBracketPick) error
	GetBracketPicks(ctx context.Context, userID string) ([]*UserBracketPick, error)
	GetBracketPicksByMatch(ctx context.Context, matchID int64) ([]*UserBracketPick, error) // for scoring
	GetChampionPick(ctx context.Context, userID string) (*string, error)
	GetUserProgressCounts(ctx context.Context, userID string) (PickemProgressCounts, error)
	GetLockedGroupCodes(ctx context.Context, userID string) ([]string, error)
	// SetGroupLocks replaces the user's set of locked groups with exactly lockedCodes
	// (groups not listed are unlocked). Lock state is sent declaratively with each group save.
	SetGroupLocks(ctx context.Context, userID string, lockedCodes []string) error
}
