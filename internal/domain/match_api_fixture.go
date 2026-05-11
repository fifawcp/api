package domain

import "context"

type MatchAPIFixtureRepository interface {
	GetByMatchID(ctx context.Context, matchID int64) (int64, error)
	UpsertFixtureID(ctx context.Context, matchID, apiFixtureID int64) error
}
