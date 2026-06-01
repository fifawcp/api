package domain

import "context"

type AwardType string

const (
	AwardGoldenBoot  AwardType = "golden_boot"
	AwardGoldenBall  AwardType = "golden_ball"
	AwardGoldenGlove AwardType = "golden_glove"
	AwardYoungPlayer AwardType = "young_player"
)

var AwardTypes = []AwardType{
	AwardGoldenBoot,
	AwardGoldenBall,
	AwardGoldenGlove,
	AwardYoungPlayer,
}

func (awardType AwardType) IsValid() bool {
	switch awardType {
	case AwardGoldenBoot, AwardGoldenBall, AwardGoldenGlove, AwardYoungPlayer:
		return true
	}
	return false
}

type UserAwardPick struct {
	UserID    string    `json:"user_id"`
	AwardType AwardType `json:"award_type"`
	PlayerID  int64     `json:"player_id"`
}

type AwardWinner struct {
	AwardType AwardType `json:"award_type"`
	PlayerID  int64     `json:"player_id"`
}

type UserAwards struct {
	Picks    []ResolvedAwardPick `json:"picks"`
	Progress StepProgress        `json:"progress"`
	IsLocked bool                `json:"is_locked"`
}

type ResolvedAwardPick struct {
	AwardType AwardType `json:"award_type"`
	Player    *Player   `json:"player"`
}

// PopularAwardPick is one entry in the "most-picked" ranking for a given award.
// PicksCount is the running tally across all users (zero for unpicked players
// that still surface because they match the award's eligibility filter).
type PopularAwardPick struct {
	Player     *Player `json:"player"`
	PicksCount int     `json:"picks_count"`
}

// PopularPicksByAward groups the popular ranking per award type. Keys mirror
// the AwardTypes slice for canonical iteration on the frontend.
type PopularPicksByAward map[AwardType][]PopularAwardPick

type AwardPickRepository interface {
	GetAwardPicks(ctx context.Context, userID string) ([]*UserAwardPick, error)
	UpsertAwardPicks(ctx context.Context, userID string, picks []*UserAwardPick) error
	GetAwardPicksByPlayer(ctx context.Context, awardType AwardType, playerID int64) ([]*UserAwardPick, error)
	GetPopularPicks(ctx context.Context, awardType AwardType, limit int, youngPlayerMaxAge int) ([]PopularAwardPick, error)
	UpsertAwardWinners(ctx context.Context, winners []*AwardWinner) error
	GetAwardWinners(ctx context.Context) ([]*AwardWinner, error)
}
