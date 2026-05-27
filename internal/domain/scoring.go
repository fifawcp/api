package domain

import (
	"context"
	"time"
)

type ScoreSourceType string

const (
	ScoreSourceGroupStandingPick ScoreSourceType = "group_standing_pick"
	ScoreSourceBestThirdPick     ScoreSourceType = "best_third_pick"
	ScoreSourceBracketPick       ScoreSourceType = "bracket_pick"
	ScoreSourceMatchScorePick    ScoreSourceType = "match_score_pick"
)

type ScoreEvent struct {
	ID         int64
	UserID     string
	SourceType ScoreSourceType
	SourceRef  string
	Points     int
	CreatedAt  time.Time
}

type ScoreEventRepository interface {
	BatchUpsertScoreEvents(ctx context.Context, events []*ScoreEvent) error
}
