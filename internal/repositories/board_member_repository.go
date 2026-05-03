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
    )
    INSERT INTO board_members (board_id, user_id, role)
    SELECT id, $2, 'member' FROM board
    WHERE EXISTS(SELECT 1 FROM board)
    RETURNING board_id
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

func (r *BoardMemberRepository) LeaveBoard(
	ctx context.Context,
	boardID string,
	userID string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourceBoardMember)
	}
	defer tx.Rollback()

	// Validate if the user is the owner and if there are other members
	query := `
		SELECT
			b.owner_user_id,
			COUNT(all_bm.user_id) AS members_count
		FROM boards b
		LEFT JOIN board_members all_bm
			ON all_bm.board_id = b.id
		WHERE b.id = $1
		GROUP BY b.owner_user_id
	`

	var ownerUserID string
	var membersCount int

	if err := tx.QueryRowContext(ctx, query, boardID).Scan(
		&ownerUserID,
		&membersCount,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrBoardNotFound
		}
		return handleDBError(err, resourceBoardMember)
	}

	// If the user is the owner, check if there are other members
	if ownerUserID == userID {
		// If there are other members, do not allow the owner to leave
		if membersCount > 1 {
			return domain.ErrBoardOwnerCannotLeaveWithMembers
		}

		// If there are no other members, delete the board
		if _, err := tx.ExecContext(ctx, `DELETE FROM boards WHERE id = $1`, boardID); err != nil {
			return handleDBError(err, resourceBoard)
		}

		return tx.Commit()
	}

	// If the user is not the owner, delete the member from the board
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM board_members WHERE board_id = $1 AND user_id = $2`,
		boardID,
		userID,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	return tx.Commit()
}
