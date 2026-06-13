package repositories

import (
	"context"
	"database/sql"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/lib/pq"
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

func (r *MatchScorePickRepository) GetMatchScorePicksByUserAndMatches(ctx context.Context, userID string, matchIDs []int64) ([]*domain.UserMatchScorePick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		user_id,
		match_id,
		home_score,
		away_score
	FROM user_match_score_picks
	WHERE user_id = $1 AND match_id = ANY($2)
	ORDER BY match_id`

	rows, err := r.db.QueryContext(ctx, query, userID, pq.Array(matchIDs))
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

func (r *MatchScorePickRepository) GetBoardMembersMatchPicks(ctx context.Context, boardID, matchID int64) ([]*domain.BoardMemberMatchPick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT
			bm.user_id,
			u.username,
			u.first_name,
			u.last_name,
			bm.role,
			bm.created_at,
			p.home_score,
			p.away_score
		FROM board_members bm
		JOIN users u ON u.id = bm.user_id
		LEFT JOIN user_match_score_picks p
			ON p.user_id = bm.user_id AND p.match_id = $2
		WHERE bm.board_id = $1
		ORDER BY bm.created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, boardID, matchID)
	if err != nil {
		return nil, handleDBError(err, resourceMatchScorePick)
	}
	defer rows.Close()

	memberPicks := []*domain.BoardMemberMatchPick{}
	for rows.Next() {
		var memberPick domain.BoardMemberMatchPick
		var homeScore, awayScore sql.NullInt64

		if err := rows.Scan(
			&memberPick.Member.UserID,
			&memberPick.Member.UserName,
			&memberPick.Member.FirstName,
			&memberPick.Member.LastName,
			&memberPick.Member.Role,
			&memberPick.Member.JoinedAt,
			&homeScore,
			&awayScore,
		); err != nil {
			return nil, handleDBError(err, resourceMatchScorePick)
		}

		if homeScore.Valid {
			score := int(homeScore.Int64)
			memberPick.HomeScore = &score
		}
		if awayScore.Valid {
			score := int(awayScore.Int64)
			memberPick.AwayScore = &score
		}

		memberPicks = append(memberPicks, &memberPick)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceMatchScorePick)
	}

	return memberPicks, nil
}
