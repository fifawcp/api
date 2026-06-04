package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
)

type BoardMemberRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewBoardMemberRepository(
	db *sql.DB,
	cfg *config.Config,
) *BoardMemberRepository {
	return &BoardMemberRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *BoardMemberRepository) CreateBoardMember(
	ctx context.Context,
	joinCode string,
	userID string,
) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, handleDBError(err, resourceBoardMember)
	}
	defer tx.Rollback()

	var boardID int64
	err = tx.QueryRowContext(ctx,
		`WITH board AS (
			SELECT id FROM boards WHERE join_code = $1
		)
		INSERT INTO board_members (board_id, user_id, role)
		SELECT id, $2, 'member' FROM board
		WHERE EXISTS(SELECT 1 FROM board)
		RETURNING board_id`,
		joinCode, userID,
	).Scan(&boardID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, domain.ErrBoardInvalidJoinCode
		}
		return 0, handleDBError(err, resourceBoardMember)
	}

	if err := r.initCompetitionScoresForMember(ctx, tx, boardID, userID); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, handleDBError(err, resourceBoardMember)
	}

	return boardID, nil
}

func (r *BoardMemberRepository) initCompetitionScoresForMember(
	ctx context.Context,
	tx *sql.Tx,
	boardID int64,
	userID string,
) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO competition_pickem_scores (
			competition_id, user_id,
			total_points, group_exact_positions, group_qualifier_hits, best_third_hits, bracket_hits
		)
		SELECT
			c.id,
			$2,
			COALESCE(agg.total_points, 0),
			COALESCE(agg.group_exact_count, 0),
			COALESCE(agg.group_qualifier_count, 0),
			COALESCE(agg.best_third_hits_count, 0),
			COALESCE(agg.bracket_hits_count, 0)
		FROM competitions c
		LEFT JOIN (
			SELECT
				SUM(se.points)                                                                         AS total_points,
				COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $3)::int AS group_exact_count,
				COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $4)::int AS group_qualifier_count,
				COUNT(*) FILTER (WHERE se.source_type = 'best_third_pick')::int                        AS best_third_hits_count,
				COUNT(*) FILTER (WHERE se.source_type = 'bracket_pick')::int                           AS bracket_hits_count
			FROM score_events se
			WHERE se.user_id = $2
			  AND se.source_type IN ('group_standing_pick', 'best_third_pick', 'bracket_pick')
		) agg ON TRUE
		WHERE c.board_id = $1 AND c.type = 'pickem'`,
		boardID, userID,
		r.cfg.Scoring.GroupPositionExact, r.cfg.Scoring.GroupQualifies,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	// Match aggregation depends on each competition's scope (stages + optional
	// team filter), so the LATERAL subquery re-evaluates per competition
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO competition_match_scores (
			competition_id, user_id,
			total_points, exact_hits, correct_outcomes
		)
		SELECT
			c.id,
			$2,
			COALESCE(agg.match_score_points, 0),
			COALESCE(agg.exact_hits_count, 0),
			COALESCE(agg.correct_outcomes_count, 0)
		FROM competitions c
		LEFT JOIN LATERAL (
			SELECT
				SUM(se.points)                                           AS match_score_points,
				COUNT(*) FILTER (WHERE se.points >= $3)                  AS exact_hits_count,
				COUNT(*) FILTER (WHERE se.points > 0 AND se.points < $3) AS correct_outcomes_count
			FROM score_events se
			INNER JOIN matches m ON m.id = se.source_ref::bigint
			INNER JOIN competition_scope_stages css
				ON css.competition_id = c.id AND css.stage = m.stage_code
			WHERE se.user_id = $2
			  AND se.source_type = 'match_score_pick'
			  AND (
				NOT EXISTS (SELECT 1 FROM competition_scope_teams cst WHERE cst.competition_id = c.id)
				OR EXISTS (
					SELECT 1 FROM competition_scope_teams cst
					WHERE cst.competition_id = c.id
					  AND cst.team_fifa_code IN (m.home_team_fifa_code, m.away_team_fifa_code)
				)
			  )
		) agg ON TRUE
		WHERE c.board_id = $1 AND c.type = 'match'`,
		boardID, userID,
		r.cfg.Scoring.MatchScoreExact,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO competition_match_scores (
			competition_id, user_id,
			total_points, exact_hits, correct_outcomes
		)
		SELECT
			c.id,
			$2,
			COALESCE(agg.match_score_points, 0),
			COALESCE(agg.exact_hits_count, 0),
			COALESCE(agg.correct_outcomes_count, 0)
		FROM competitions c
		LEFT JOIN LATERAL (
			SELECT
				SUM(se.points)                                           AS match_score_points,
				COUNT(*) FILTER (WHERE se.points >= $3)                  AS exact_hits_count,
				COUNT(*) FILTER (WHERE se.points > 0 AND se.points < $3) AS correct_outcomes_count
			FROM score_events se
			WHERE se.user_id = $2
			  AND se.source_type = 'match_score_pick'
			  AND se.source_ref::bigint = c.match_id
		) agg ON TRUE
		WHERE c.board_id = $1 AND c.type = 'pool'`,
		boardID, userID,
		r.cfg.Scoring.MatchScoreExact,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	return nil
}

func (r *BoardMemberRepository) GetBoardMember(
	ctx context.Context,
	boardID int64,
	userID string,
) (*domain.BoardMember, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT board_id, user_id, role, created_at
		FROM board_members
		WHERE board_id = $1 AND user_id = $2`

	var boardMember domain.BoardMember

	if err := r.db.QueryRowContext(ctx, query, boardID, userID).Scan(
		&boardMember.BoardID,
		&boardMember.UserID,
		&boardMember.Role,
		&boardMember.CreatedAt,
	); err != nil {
		return nil, handleDBError(err, resourceBoardMember)
	}

	return &boardMember, nil
}

func (r *BoardMemberRepository) UpdateBoardMemberRole(
	ctx context.Context,
	boardID int64,
	userID string,
	role domain.BoardMemberRole,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}
	defer tx.Rollback()

	var currentRole domain.BoardMemberRole
	err = tx.QueryRowContext(ctx,
		`SELECT role FROM board_members
		WHERE board_id = $1 AND user_id = $2 FOR UPDATE`,
		boardID,
		userID,
	).Scan(&currentRole)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrBoardMemberNotFound
		}

		return handleDBError(err, resourceBoardMember)
	}

	if currentRole == domain.BoardMemberRoleOwner {
		return domain.ErrForbidden
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE board_members SET role = $1
		WHERE board_id = $2 AND user_id = $3`,
		role,
		boardID,
		userID,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	return tx.Commit()
}

func (r *BoardMemberRepository) TransferOwnership(
	ctx context.Context,
	boardID int64,
	oldOwnerUserID, newOwnerUserID string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}
	defer tx.Rollback()

	demoteResult, err := tx.ExecContext(ctx,
		`UPDATE board_members SET role = $3
		 WHERE board_id = $1 AND user_id = $2 AND role = 'owner'`,
		boardID, oldOwnerUserID, domain.BoardMemberRoleAdmin,
	)
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	demotedRows, err := demoteResult.RowsAffected()
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}
	if demotedRows == 0 {
		return domain.ErrForbidden
	}

	promoteResult, err := tx.ExecContext(ctx,
		`UPDATE board_members SET role = $3
		 WHERE board_id = $1 AND user_id = $2`,
		boardID, newOwnerUserID, domain.BoardMemberRoleOwner,
	)
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}
	promotedRows, err := promoteResult.RowsAffected()
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}
	if promotedRows == 0 {
		return domain.ErrBoardMemberNotFound
	}

	return tx.Commit()
}

func (r *BoardMemberRepository) RemoveBoardMember(
	ctx context.Context,
	boardID int64,
	userID string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}
	defer tx.Rollback()

	var currentRole domain.BoardMemberRole
	err = tx.QueryRowContext(ctx,
		`SELECT role FROM board_members
		WHERE board_id = $1 AND user_id = $2 FOR UPDATE`,
		boardID,
		userID,
	).Scan(&currentRole)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrBoardMemberNotFound
		}

		return handleDBError(err, resourceBoardMember)
	}

	if currentRole == domain.BoardMemberRoleOwner {
		return domain.ErrForbidden
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM board_members WHERE board_id = $1 AND user_id = $2`,
		boardID,
		userID,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	if err := r.deleteCompetitionScoresForMember(ctx, tx, boardID, userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *BoardMemberRepository) LeaveBoard(
	ctx context.Context,
	boardID int64,
	userID string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}
	defer tx.Rollback()

	var role domain.BoardMemberRole
	var membersCount int
	if err := tx.QueryRowContext(ctx,
		`SELECT
			bm.role,
			COUNT(all_bm.user_id) AS members_count
		FROM board_members bm
		LEFT JOIN board_members all_bm ON all_bm.board_id = bm.board_id
		WHERE bm.board_id = $1 AND bm.user_id = $2
		GROUP BY bm.role`,
		boardID, userID,
	).Scan(&role, &membersCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrBoardMemberNotFound
		}
		return handleDBError(err, resourceBoardMember)
	}

	if role != domain.BoardMemberRoleOwner {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM board_members WHERE board_id = $1 AND user_id = $2`,
			boardID, userID,
		); err != nil {
			return handleDBError(err, resourceBoardMember)
		}
		if err := r.deleteCompetitionScoresForMember(ctx, tx, boardID, userID); err != nil {
			return err
		}
		return tx.Commit()
	}

	// Owner leaving alone — delete the entire board (cascades to members + competitions)
	if membersCount == 1 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM boards WHERE id = $1`, boardID); err != nil {
			return handleDBError(err, resourceBoard)
		}
		return tx.Commit()
	}

	// Owner leaving with others — pick a successor and rotate the role
	var successorUserID string
	if err := tx.QueryRowContext(ctx,
		`SELECT user_id FROM board_members
		 WHERE board_id = $1 AND user_id != $2
		 ORDER BY (role = 'admin') DESC, created_at ASC, user_id ASC
		 LIMIT 1
		 FOR UPDATE`,
		boardID, userID,
	).Scan(&successorUserID); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM board_members WHERE board_id = $1 AND user_id = $2`,
		boardID, userID,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE board_members SET role = $3
		 WHERE board_id = $1 AND user_id = $2`,
		boardID, successorUserID, domain.BoardMemberRoleOwner,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	if err := r.deleteCompetitionScoresForMember(ctx, tx, boardID, userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *BoardMemberRepository) deleteCompetitionScoresForMember(
	ctx context.Context,
	tx *sql.Tx,
	boardID int64,
	userID string,
) error {
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM competition_pickem_scores
		 WHERE user_id = $2
		   AND competition_id IN (SELECT id FROM competitions WHERE board_id = $1)`,
		boardID, userID,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM competition_match_scores
		 WHERE user_id = $2
		   AND competition_id IN (SELECT id FROM competitions WHERE board_id = $1)`,
		boardID, userID,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	return nil
}
