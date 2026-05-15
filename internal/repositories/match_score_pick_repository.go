package repositories

import (
	"context"
	"database/sql"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
)

type MatchScorePickRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewMatchScorePickRepository(db *sql.DB, cfg *config.Config) *MatchScorePickRepository {
	return &MatchScorePickRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *MatchScorePickRepository) UpsertMatchScorePick(ctx context.Context, pick *domain.UserMatchScorePick) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	// Idempotent upsert - if the pick already exists, it will be updated
	query := `INSERT INTO user_match_score_picks (
		user_id,
		match_id,
		home_score,
		away_score
	) VALUES ($1, $2, $3, $4)
	ON CONFLICT (user_id, match_id) DO UPDATE
	SET
		home_score = EXCLUDED.home_score,
	  away_score = EXCLUDED.away_score`

	if _, err := r.db.ExecContext(
		ctx,
		query,
		pick.UserID,
		pick.MatchID,
		pick.HomeScore,
		pick.AwayScore,
	); err != nil {
		return handleDBError(err, resourceMatchScorePick)
	}

	return nil
}

func (r *MatchScorePickRepository) GetMatchScorePicksByUser(ctx context.Context, userID string) ([]*domain.UserMatchScorePick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		user_id,
		match_id,
		home_score,
		away_score
	FROM user_match_score_picks
	WHERE user_id = $1 ORDER BY match_id`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, handleDBError(err, resourceMatchScorePick)
	}
	defer rows.Close()

	picks := []*domain.UserMatchScorePick{}
	for rows.Next() {
		var pick domain.UserMatchScorePick
		if err := rows.Scan(
			&pick.UserID,
			&pick.MatchID,
			&pick.HomeScore,
			&pick.AwayScore,
		); err != nil {
			return nil, handleDBError(err, resourceMatchScorePick)
		}

		picks = append(picks, &pick)
	}

	return picks, rows.Err()
}

func (r *MatchScorePickRepository) CountMatchScorePicksByUser(ctx context.Context, userID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	var count int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_match_score_picks WHERE user_id = $1`,
		userID,
	).Scan(&count); err != nil {
		return 0, handleDBError(err, resourceMatchScorePick)
	}

	return count, nil
}

func (r *MatchScorePickRepository) GetMatchScorePicksByMatch(ctx context.Context, matchID int64) ([]*domain.UserMatchScorePick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		user_id,
		match_id,
		home_score,
		away_score
	FROM user_match_score_picks WHERE match_id = $1`

	rows, err := r.db.QueryContext(ctx, query, matchID)
	if err != nil {
		return nil, handleDBError(err, resourceMatchScorePick)
	}
	defer rows.Close()

	picks := []*domain.UserMatchScorePick{}
	for rows.Next() {
		var pick domain.UserMatchScorePick
		if err := rows.Scan(&pick.UserID, &pick.MatchID, &pick.HomeScore, &pick.AwayScore); err != nil {
			return nil, handleDBError(err, resourceMatchScorePick)
		}

		picks = append(picks, &pick)
	}

	return picks, rows.Err()
}
