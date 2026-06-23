package football

import "context"

// FixtureFetcher is the slice of the football client that match-result-sync
// consumers (the cron poller and the manual admin sync) depend on. It lives
// here, next to the client that implements it, so both the services and jobs
// packages can reference it without creating an import cycle between them.
type FixtureFetcher interface {
	GetFixture(ctx context.Context, fixtureID int64) (*FixtureResponse, error)
	GetFixturesByTeam(ctx context.Context, teamAPIID int64) ([]FixtureResponse, error)
}
