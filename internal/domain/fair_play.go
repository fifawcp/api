package domain

import "context"

type MatchFairPlay struct {
	MatchID                     int64
	TeamFIFACode                string
	YellowCards                 int
	IndirectRedCards            int
	DirectRedCards              int
	YellowCardAndDirectRedCards int
}

type MatchFairPlayRepository interface {
	Upsert(ctx context.Context, records []MatchFairPlay) error
	GetFairPlayTotalsByGroup(ctx context.Context, groupCode string) (map[string]int, error)
}
