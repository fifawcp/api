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
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH board AS (
      SELECT id FROM boards WHERE join_code = $1
    ),
    new_board_member AS (
      INSERT INTO board_members (board_id, user_id, role)
      SELECT id, $2, 'member' FROM board
      WHERE EXISTS(SELECT 1 FROM board)
      RETURNING board_id
    ),
    new_board_ranking AS (
      INSERT INTO board_rankings (board_id, user_id)
      SELECT board_id, $2 FROM new_board_member
    )
    SELECT board_id FROM new_board_member
	`

	var boardID string

	err := r.db.QueryRowContext(ctx, query, joinCode, userID).Scan(&boardID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrBoardInvalidJoinCode
		}
		return handleDBError(err, resourceBoardMember)
	}

	return nil
}

func (r *BoardMemberRepository) GetBoardMember(
	ctx context.Context,
	boardID string,
	userID string,
) (*domain.BoardMember, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		board_id,
		user_id,
		role,
		created_at
	FROM board_members
	WHERE board_id = $1 AND user_id = $2`

	var boardMember domain.BoardMember

	if err := r.db.QueryRowContext(
		ctx,
		query,
		boardID,
		userID,
	).Scan(
		&boardMember.BoardID,
		&boardMember.UserID,
		&boardMember.Role,
		&boardMember.CreatedAt,
	); err != nil {
		return nil, handleDBError(err, resourceBoardMember)
	}

	return &boardMember, nil
}

func (r *BoardMemberRepository) GetBoardMembers(
	ctx context.Context,
	boardID string,
) ([]*domain.BoardMember, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		bm.board_id,
		bm.user_id,
		bm.role,
		bm.created_at,
		u.username
	FROM board_members bm
	JOIN users u ON bm.user_id = u.id
	WHERE bm.board_id = $1
	ORDER BY bm.created_at DESC
	`

	var boardMembers []*domain.BoardMember

	rows, err := r.db.QueryContext(ctx, query, boardID)
	if err != nil {
		return nil, handleDBError(err, resourceBoardMember)
	}
	defer rows.Close()

	for rows.Next() {
		var boardMember domain.BoardMember
		if err := rows.Scan(
			&boardMember.BoardID,
			&boardMember.UserID,
			&boardMember.Role,
			&boardMember.CreatedAt,
			&boardMember.UserName,
		); err != nil {
			return nil, handleDBError(err, resourceBoardMember)
		}
		boardMembers = append(boardMembers, &boardMember)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceBoardMember)
	}

	return boardMembers, nil
}

func (r *BoardMemberRepository) UpdateBoardMemberRole(
	ctx context.Context,
	boardID string,
	userID string,
	role domain.BoardMemberRole,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	// Check if member exists and if they're the owner
	checkQuery := `
  SELECT 
    CASE 
      WHEN NOT EXISTS (
        SELECT 1 FROM board_members WHERE board_id = $1 AND user_id = $2
      ) THEN 'MEMBER_NOT_FOUND'
      WHEN EXISTS (
        SELECT 1 FROM boards WHERE id = $1 AND owner_user_id = $2
      ) THEN 'IS_OWNER'
      ELSE 'CAN_UPDATE'
    END as state
  `

	var state string
	err := r.db.QueryRowContext(ctx, checkQuery, boardID, userID).Scan(&state)
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	switch state {
	case "MEMBER_NOT_FOUND":
		return domain.ErrBoardMemberNotFound
	case "IS_OWNER":
		return domain.ErrForbidden
	}

	// Perform update
	updateQuery := `UPDATE board_members SET role = $1 WHERE board_id = $2 AND user_id = $3`
	_, err = r.db.ExecContext(ctx, updateQuery, role, boardID, userID)
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	return nil
}

func (r *BoardMemberRepository) RemoveBoardMember(
	ctx context.Context,
	boardID string,
	userID string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	// Check if member exists and if they're the owner
	checkQuery := `
  SELECT 
    CASE 
      WHEN NOT EXISTS (
        SELECT 1 FROM board_members WHERE board_id = $1 AND user_id = $2
      ) THEN 'MEMBER_NOT_FOUND'
      WHEN EXISTS (
        SELECT 1 FROM boards WHERE id = $1 AND owner_user_id = $2
      ) THEN 'IS_OWNER'
      ELSE 'CAN_DELETE'
    END as state
  `

	var state string
	err := r.db.QueryRowContext(ctx, checkQuery, boardID, userID).Scan(&state)
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	switch state {
	case "MEMBER_NOT_FOUND":
		return domain.ErrBoardMemberNotFound
	case "IS_OWNER":
		return domain.ErrForbidden
	}

	// Perform delete
	deleteQuery := `DELETE FROM board_members WHERE board_id = $1 AND user_id = $2`
	_, err = r.db.ExecContext(ctx, deleteQuery, boardID, userID)
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	return nil
}
