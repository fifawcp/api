package repositories

import (
	"context"
	"database/sql"

	"github.com/fifawcp/api/internal/infrastructure/config"
)

type MatchAPIFixtureRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewMatchAPIFixtureRepository(db *sql.DB, cfg *config.Config) *MatchAPIFixtureRepository {
	return &MatchAPIFixtureRepository{db: db, cfg: cfg}
}

func (r *MatchAPIFixtureRepository) GetByMatchID(ctx context.Context, matchID int64) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT api_fixture_id FROM match_api_fixtures WHERE match_id = $1`

	var apiFixtureID int64
	err := r.db.QueryRowContext(ctx, query, matchID).Scan(&apiFixtureID)
	if err != nil {
		return 0, handleDBError(err, resourceMatchAPIFixture)
	}

	return apiFixtureID, nil
}

func (r *MatchAPIFixtureRepository) UpsertFixtureID(ctx context.Context, matchID, apiFixtureID int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `INSERT INTO match_api_fixtures (match_id, api_fixture_id)
		VALUES ($1, $2)
		ON CONFLICT (match_id) DO UPDATE SET api_fixture_id = EXCLUDED.api_fixture_id`

	_, err := r.db.ExecContext(ctx, query, matchID, apiFixtureID)

	return handleDBError(err, resourceMatchAPIFixture)
}
