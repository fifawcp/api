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

type AwardPickRepository interface {
	GetAwardPicks(ctx context.Context, userID string) ([]*UserAwardPick, error)
	UpsertAwardPicks(ctx context.Context, userID string, picks []*UserAwardPick) error
	GetAwardPicksByPlayer(ctx context.Context, awardType AwardType, playerID int64) ([]*UserAwardPick, error)
	UpsertAwardWinners(ctx context.Context, winners []*AwardWinner) error
	GetAwardWinners(ctx context.Context) ([]*AwardWinner, error)
}
