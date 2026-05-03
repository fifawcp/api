package repositories

import (
	"context"
	"database/sql"

	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/lib/pq"
)

type UserScoreRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewUserScoreRepository(
	db *sql.DB,
	cfg *config.Config,
) *UserScoreRepository {
	return &UserScoreRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *UserScoreRepository) BatchUpdateUserScores(
	ctx context.Context,
	userIDs []string,
	exactScorePts int,
) error {
	if len(userIDs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	// Recompute the user scores from the score events
	query := `
		UPDATE user_scores us
		SET
			pickem_points      = s.pickem_pts,
			match_score_points = s.match_score_pts,
			total_points       = s.pickem_pts + s.match_score_pts,
			exact_hits         = s.exact_hits_cnt,
			correct_outcomes   = s.correct_outcomes_cnt,
			updated_at         = NOW()
		FROM (
			SELECT
				user_id,
				COALESCE(SUM(points) FILTER (WHERE source_type IN ('group_standing_pick','best_third_pick','bracket_pick')), 0) AS pickem_pts,
				COALESCE(SUM(points) FILTER (WHERE source_type = 'match_score_pick'), 0)                                        AS match_score_pts,
				COUNT(*)             FILTER (WHERE source_type = 'match_score_pick' AND points >= $2)                           AS exact_hits_cnt,
				COUNT(*)             FILTER (WHERE source_type = 'match_score_pick' AND points > 0)                             AS correct_outcomes_cnt
			FROM score_events
			WHERE user_id = ANY($1::uuid[])
			GROUP BY user_id
		) s
		WHERE us.user_id = s.user_id
	`

	if _, err := r.db.ExecContext(ctx, query, pq.Array(userIDs), exactScorePts); err != nil {
		return handleDBError(err, resourceUserScore)
	}

	return nil
}
