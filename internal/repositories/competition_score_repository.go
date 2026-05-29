package repositories

import (
	"context"
	"database/sql"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/lib/pq"
)

type CompetitionScoreRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewCompetitionScoreRepository(db *sql.DB, cfg *config.Config) *CompetitionScoreRepository {
	return &CompetitionScoreRepository{db: db, cfg: cfg}
}

func (r *CompetitionScoreRepository) FindMatchCompetitionsByMatches(
	ctx context.Context,
	matchIDs []int64,
) ([]int64, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT DISTINCT c.id
		FROM competitions c
		INNER JOIN competition_scope_stages css ON css.competition_id = c.id
		INNER JOIN matches m ON m.stage_code = css.stage AND m.id = ANY($1::bigint[])
		WHERE c.type = 'match'
		  AND (
			NOT EXISTS (SELECT 1 FROM competition_scope_teams cst WHERE cst.competition_id = c.id)
			OR EXISTS (
				SELECT 1 FROM competition_scope_teams cst
				WHERE cst.competition_id = c.id
				  AND cst.team_fifa_code IN (m.home_team_fifa_code, m.away_team_fifa_code)
			)
		  )
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(matchIDs))
	if err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, handleDBError(err, resourceCompetitionScore)
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func (r *CompetitionScoreRepository) BatchUpsertMatchScores(
	ctx context.Context,
	competitionID int64,
	userIDs []string,
	exactScorePts int,
) error {
	if len(userIDs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH scope_matches AS (
			SELECT m.id
			FROM matches m
			INNER JOIN competition_scope_stages css ON css.competition_id = $1 AND css.stage = m.stage_code
			WHERE (
				NOT EXISTS (SELECT 1 FROM competition_scope_teams cst WHERE cst.competition_id = $1)
				OR EXISTS (
					SELECT 1 FROM competition_scope_teams cst
					WHERE cst.competition_id = $1
					  AND cst.team_fifa_code IN (m.home_team_fifa_code, m.away_team_fifa_code)
				)
			)
		),
		computed AS (
			SELECT
				se.user_id,
				COALESCE(SUM(se.points), 0)                                   AS match_score_points,
				COUNT(*) FILTER (WHERE se.points >= $3)                       AS exact_hits_count,
				COUNT(*) FILTER (WHERE se.points > 0 AND se.points < $3)      AS correct_outcomes_count
			FROM score_events se
			INNER JOIN scope_matches sm ON sm.id = se.source_ref::bigint
			WHERE se.user_id = ANY($2::uuid[])
			  AND se.source_type = 'match_score_pick'
			GROUP BY se.user_id
		)
		INSERT INTO competition_match_scores (
			competition_id, user_id, total_points,
			exact_hits, correct_outcomes, updated_at
		)
		SELECT $1, c.user_id, c.match_score_points, c.exact_hits_count, c.correct_outcomes_count, NOW()
		FROM computed c
		WHERE EXISTS (
			SELECT 1 FROM competitions co
			INNER JOIN board_members bm ON bm.board_id = co.board_id AND bm.user_id = c.user_id
			WHERE co.id = $1
		)
		ON CONFLICT (competition_id, user_id) DO UPDATE SET
			total_points     = EXCLUDED.total_points,
			exact_hits       = EXCLUDED.exact_hits,
			correct_outcomes = EXCLUDED.correct_outcomes,
			updated_at       = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query, competitionID, pq.Array(userIDs), exactScorePts)
	if err != nil {
		return handleDBError(err, resourceCompetitionScore)
	}

	return nil
}

func (r *CompetitionScoreRepository) BatchUpsertPickemScores(
	ctx context.Context,
	competitionIDs []int64,
	userIDs []string,
) error {
	if len(competitionIDs) == 0 || len(userIDs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	// Recompute per user from score_events, then fan-out across competition IDs
	query := `
		WITH computed AS (
			SELECT
				se.user_id,
				COALESCE(SUM(se.points), 0)                                                            AS total_points,
				COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $3)::int AS group_exact_count,
				COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $4)::int AS group_qualifier_count,
				COUNT(*) FILTER (WHERE se.source_type = 'best_third_pick')::int                        AS best_third_hits_count,
				COUNT(*) FILTER (WHERE se.source_type = 'bracket_pick')::int                           AS bracket_hits_count,
				COUNT(*) FILTER (WHERE se.source_type = 'award_pick')::int                             AS award_hits_count
			FROM score_events se
			WHERE se.user_id = ANY($2::uuid[])
			  AND se.source_type IN ('group_standing_pick', 'best_third_pick', 'bracket_pick', 'award_pick')
			GROUP BY se.user_id
		)
		INSERT INTO competition_pickem_scores (
			competition_id, user_id, total_points,
			group_exact_positions, group_qualifier_hits, best_third_hits, bracket_hits, award_hits, updated_at
		)
		SELECT
			comp_id,
			c.user_id,
			c.total_points,
			c.group_exact_count,
			c.group_qualifier_count,
			c.best_third_hits_count,
			c.bracket_hits_count,
			c.award_hits_count,
			NOW()
		FROM computed c
		CROSS JOIN UNNEST($1::bigint[]) AS comp_id
		WHERE EXISTS (
			SELECT 1 FROM competitions co
			INNER JOIN board_members bm ON bm.board_id = co.board_id AND bm.user_id = c.user_id
			WHERE co.id = comp_id
		)
		ON CONFLICT (competition_id, user_id) DO UPDATE SET
			total_points          = EXCLUDED.total_points,
			group_exact_positions = EXCLUDED.group_exact_positions,
			group_qualifier_hits  = EXCLUDED.group_qualifier_hits,
			best_third_hits       = EXCLUDED.best_third_hits,
			bracket_hits          = EXCLUDED.bracket_hits,
			award_hits            = EXCLUDED.award_hits,
			updated_at            = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		pq.Array(competitionIDs), pq.Array(userIDs),
		r.cfg.Scoring.GroupPositionExact, r.cfg.Scoring.GroupQualifies,
	)
	if err != nil {
		return handleDBError(err, resourceCompetitionScore)
	}

	return nil
}

func (r *CompetitionScoreRepository) GetLeaderboard(
	ctx context.Context,
	competitionID int64,
	page, limit int,
	q string,
) (*domain.CompetitionLeaderboardPage, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	var competitionType domain.CompetitionType
	if err := r.db.QueryRowContext(ctx,
		`SELECT type FROM competitions WHERE id = $1`,
		competitionID,
	).Scan(&competitionType); err != nil {
		return nil, handleDBError(err, resourceCompetition)
	}

	switch competitionType {
	case domain.CompetitionTypePickem:
		return r.getPickemLeaderboard(ctx, competitionID, page, limit, q)
	case domain.CompetitionTypeMatch:
		return r.getMatchLeaderboard(ctx, competitionID, page, limit, q)
	default:
		return nil, domain.ErrCompetitionNotFound
	}
}

func (r *CompetitionScoreRepository) getPickemLeaderboard(
	ctx context.Context,
	competitionID int64,
	page, limit int,
	q string,
) (*domain.CompetitionLeaderboardPage, error) {
	offset := (page - 1) * limit

	query := `
		WITH ranked AS (
			SELECT
				cps.user_id,
				u.username,
				u.first_name,
				u.last_name,
				bm.role,
				bm.created_at AS joined_at,
				cps.total_points,
				cps.group_exact_positions,
				cps.group_qualifier_hits,
				cps.best_third_hits,
				cps.bracket_hits,
				cps.award_hits,
				RANK() OVER (
					ORDER BY
						cps.total_points          DESC,
						cps.bracket_hits          DESC,
						cps.award_hits            DESC,
						cps.best_third_hits       DESC,
						cps.group_exact_positions DESC,
						cps.group_qualifier_hits  DESC,
						bm.created_at ASC,
						cps.user_id ASC
				) AS rank
			FROM competition_pickem_scores cps
			INNER JOIN competitions comp ON comp.id = cps.competition_id
			INNER JOIN users u           ON u.id = cps.user_id
			INNER JOIN board_members bm  ON bm.board_id = comp.board_id AND bm.user_id = cps.user_id
			WHERE cps.competition_id = $1
		),
		filtered AS (
			SELECT *, COUNT(*) OVER () AS total
			FROM ranked
			WHERE $4::text = ''
			   OR username   ILIKE '%' || $4 || '%'
			   OR first_name ILIKE '%' || $4 || '%'
			   OR last_name  ILIKE '%' || $4 || '%'
		)
		SELECT
			user_id, username, first_name, last_name, role, joined_at,
			rank, total_points,
			group_exact_positions, group_qualifier_hits, best_third_hits, bracket_hits, award_hits,
			total
		FROM filtered
		ORDER BY rank ASC, joined_at ASC, user_id ASC
		OFFSET $2 LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, competitionID, offset, limit, q)
	if err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}
	defer rows.Close()

	leaderboard := &domain.CompetitionLeaderboardPage{
		Members:    []*domain.CompetitionLeaderboardEntry{},
		Pagination: domain.Pagination{Page: page, Limit: limit},
	}

	for rows.Next() {
		entry := &domain.CompetitionLeaderboardEntry{}
		score := &domain.PickemScore{}
		var total int

		if err := rows.Scan(
			&entry.Member.UserID,
			&entry.Member.UserName,
			&entry.Member.FirstName,
			&entry.Member.LastName,
			&entry.Member.Role,
			&entry.Member.JoinedAt,
			&entry.Rank,
			&score.Total,
			&score.GroupExactPositions,
			&score.GroupQualifierHits,
			&score.BestThirdHits,
			&score.BracketHits,
			&score.AwardHits,
			&total,
		); err != nil {
			return nil, handleDBError(err, resourceCompetitionScore)
		}

		entry.Score = score
		leaderboard.Pagination.Total = total
		leaderboard.Members = append(leaderboard.Members, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}

	if len(leaderboard.Members) == 0 {
		if err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*)
			 FROM competition_pickem_scores cps
			 INNER JOIN users u ON u.id = cps.user_id
			 WHERE cps.competition_id = $1
			   AND ($2::text = ''
			        OR u.username   ILIKE '%' || $2 || '%'
			        OR u.first_name ILIKE '%' || $2 || '%'
			        OR u.last_name  ILIKE '%' || $2 || '%')`,
			competitionID, q,
		).Scan(&leaderboard.Pagination.Total); err != nil && err != sql.ErrNoRows {
			return nil, handleDBError(err, resourceCompetitionScore)
		}
	}

	leaderboard.Pagination.HasMore = page*limit < leaderboard.Pagination.Total
	return leaderboard, nil
}

func (r *CompetitionScoreRepository) GetUserPickemStats(
	ctx context.Context,
	competitionID int64,
	userID string,
) (domain.CompetitionUserStats, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH ranked AS (
			SELECT
				cps.user_id,
				cps.total_points,
				RANK() OVER (
					ORDER BY
						cps.total_points          DESC,
						cps.bracket_hits          DESC,
						cps.award_hits            DESC,
						cps.best_third_hits       DESC,
						cps.group_exact_positions DESC,
						cps.group_qualifier_hits  DESC,
						bm.created_at ASC,
						cps.user_id ASC
				) AS rank
			FROM competition_pickem_scores cps
			INNER JOIN competitions comp ON comp.id = cps.competition_id
			INNER JOIN board_members bm  ON bm.board_id = comp.board_id AND bm.user_id = cps.user_id
			WHERE cps.competition_id = $1
		)
		SELECT rank, total_points
		FROM ranked
		WHERE user_id = $2
	`

	var stats domain.CompetitionUserStats
	err := r.db.QueryRowContext(ctx, query, competitionID, userID).Scan(&stats.Rank, &stats.Points)
	if err == sql.ErrNoRows {
		return domain.CompetitionUserStats{}, nil
	}
	if err != nil {
		return domain.CompetitionUserStats{}, handleDBError(err, resourceCompetitionScore)
	}

	return stats, nil
}

func (r *CompetitionScoreRepository) GetUserMatchStats(
	ctx context.Context,
	competitionID int64,
	userID string,
) (domain.CompetitionUserStats, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH ranked AS (
			SELECT
				cms.user_id,
				cms.total_points,
				RANK() OVER (
					ORDER BY
						cms.total_points     DESC,
						cms.exact_hits       DESC,
						cms.correct_outcomes DESC,
						bm.created_at ASC,
						cms.user_id ASC
				) AS rank
			FROM competition_match_scores cms
			INNER JOIN competitions comp ON comp.id = cms.competition_id
			INNER JOIN board_members bm  ON bm.board_id = comp.board_id AND bm.user_id = cms.user_id
			WHERE cms.competition_id = $1
		)
		SELECT rank, total_points
		FROM ranked
		WHERE user_id = $2
	`

	var stats domain.CompetitionUserStats
	err := r.db.QueryRowContext(ctx, query, competitionID, userID).Scan(&stats.Rank, &stats.Points)
	if err == sql.ErrNoRows {
		return domain.CompetitionUserStats{}, nil
	}
	if err != nil {
		return domain.CompetitionUserStats{}, handleDBError(err, resourceCompetitionScore)
	}

	return stats, nil
}

func (r *CompetitionScoreRepository) getMatchLeaderboard(
	ctx context.Context,
	competitionID int64,
	page, limit int,
	q string,
) (*domain.CompetitionLeaderboardPage, error) {
	offset := (page - 1) * limit

	query := `
		WITH ranked AS (
			SELECT
				cms.user_id,
				u.username,
				u.first_name,
				u.last_name,
				bm.role,
				bm.created_at AS joined_at,
				cms.total_points,
				cms.exact_hits,
				cms.correct_outcomes,
				RANK() OVER (
					ORDER BY
						cms.total_points     DESC,
						cms.exact_hits       DESC,
						cms.correct_outcomes DESC,
						bm.created_at ASC,
						cms.user_id ASC
				) AS rank
			FROM competition_match_scores cms
			INNER JOIN competitions comp ON comp.id = cms.competition_id
			INNER JOIN users u           ON u.id = cms.user_id
			INNER JOIN board_members bm  ON bm.board_id = comp.board_id AND bm.user_id = cms.user_id
			WHERE cms.competition_id = $1
		),
		filtered AS (
			SELECT *, COUNT(*) OVER () AS total
			FROM ranked
			WHERE $4::text = ''
			   OR username   ILIKE '%' || $4 || '%'
			   OR first_name ILIKE '%' || $4 || '%'
			   OR last_name  ILIKE '%' || $4 || '%'
		)
		SELECT
			user_id, username, first_name, last_name, role, joined_at,
			rank, total_points,
			exact_hits, correct_outcomes,
			total
		FROM filtered
		ORDER BY rank ASC, joined_at ASC, user_id ASC
		OFFSET $2 LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, competitionID, offset, limit, q)
	if err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}
	defer rows.Close()

	leaderboard := &domain.CompetitionLeaderboardPage{
		Members:    []*domain.CompetitionLeaderboardEntry{},
		Pagination: domain.Pagination{Page: page, Limit: limit},
	}

	for rows.Next() {
		entry := &domain.CompetitionLeaderboardEntry{}
		score := &domain.MatchScore{}
		var total int

		if err := rows.Scan(
			&entry.Member.UserID,
			&entry.Member.UserName,
			&entry.Member.FirstName,
			&entry.Member.LastName,
			&entry.Member.Role,
			&entry.Member.JoinedAt,
			&entry.Rank,
			&score.Total,
			&score.ExactHits,
			&score.CorrectOutcomes,
			&total,
		); err != nil {
			return nil, handleDBError(err, resourceCompetitionScore)
		}

		entry.Score = score
		leaderboard.Pagination.Total = total
		leaderboard.Members = append(leaderboard.Members, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}

	if len(leaderboard.Members) == 0 {
		if err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*)
			 FROM competition_match_scores cms
			 INNER JOIN users u ON u.id = cms.user_id
			 WHERE cms.competition_id = $1
			   AND ($2::text = ''
			        OR u.username   ILIKE '%' || $2 || '%'
			        OR u.first_name ILIKE '%' || $2 || '%'
			        OR u.last_name  ILIKE '%' || $2 || '%')`,
			competitionID, q,
		).Scan(&leaderboard.Pagination.Total); err != nil && err != sql.ErrNoRows {
			return nil, handleDBError(err, resourceCompetitionScore)
		}
	}

	leaderboard.Pagination.HasMore = page*limit < leaderboard.Pagination.Total
	return leaderboard, nil
}
