package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
)

type BoardRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewBoardRepository(db *sql.DB, cfg *config.Config) *BoardRepository {
	return &BoardRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *BoardRepository) CreateBoardWithOwner(
	ctx context.Context,
	board *domain.Board,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH new_board AS (
			INSERT INTO boards (name, owner_user_id, join_code)
			VALUES ($1, $2, $3)
			RETURNING id, owner_user_id, created_at
		),
		new_board_member AS (
			INSERT INTO board_members (board_id, user_id, role)
			SELECT id, owner_user_id, 'admin' FROM new_board
		),
		new_board_ranking AS (
			INSERT INTO board_rankings (board_id, user_id)
			SELECT id, owner_user_id FROM new_board
		)
		SELECT id, created_at FROM new_board`

	err := r.db.QueryRowContext(
		ctx,
		query,
		board.Name,
		board.OwnerUserID,
		board.JoinCode,
	).Scan(
		&board.ID,
		&board.CreatedAt,
	)
	if err != nil {
		return handleDBError(err, resourceBoard)
	}

	return nil
}

func (r *BoardRepository) GetUserBoards(ctx context.Context, userID string) ([]*domain.Board, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT
			b.id,
			b.name,
			b.owner_user_id,
			b.join_code,
			b.created_at
		FROM boards b
		INNER JOIN board_members bm ON b.id = bm.board_id
		WHERE bm.user_id = $1
		ORDER BY b.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}
	defer rows.Close()

	boards := []*domain.Board{}

	for rows.Next() {
		var board domain.Board
		err := rows.Scan(
			&board.ID,
			&board.Name,
			&board.OwnerUserID,
			&board.JoinCode,
			&board.CreatedAt,
		)
		if err != nil {
			return nil, handleDBError(err, resourceBoard)
		}

		boards = append(boards, &board)
	}

	if err = rows.Err(); err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	return boards, nil
}

func (r *BoardRepository) GetBoardByID(ctx context.Context, boardID string) (*domain.Board, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT
			id,
			name,
			owner_user_id,
			join_code,
			created_at
		FROM boards
		WHERE id = $1
	`

	var board domain.Board

	err := r.db.QueryRowContext(ctx, query, boardID).Scan(
		&board.ID,
		&board.Name,
		&board.OwnerUserID,
		&board.JoinCode,
		&board.CreatedAt,
	)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	return &board, nil
}

func (r *BoardRepository) UpdateJoinCode(ctx context.Context, boardID string, joinCode string) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		UPDATE boards
		SET join_code = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, joinCode, boardID)
	if err != nil {
		return handleDBError(err, resourceBoard)
	}

	return nil
}

func (r *BoardRepository) UpdateBoard(ctx context.Context, boardID string, board *domain.Board) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	// Only update non-zero values
	fields := []string{}
	args := []any{}
	argIndex := 1

	if board.Name != "" {
		fields = append(fields, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, board.Name)
		argIndex++
	}

	if len(fields) == 0 {
		return nil
	}

	args = append(args, boardID)

	query := fmt.Sprintf(
		"UPDATE boards SET %s WHERE id = $%d",
		strings.Join(fields, ", "),
		argIndex,
	)

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return handleDBError(err, resourceBoard)
	}

	return nil
}

func (r *BoardRepository) DeleteBoard(
	ctx context.Context,
	boardID string,
	userID string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		DELETE FROM boards
		WHERE id = $1 AND owner_user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, boardID, userID)
	if err != nil {
		return handleDBError(err, resourceBoard)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleDBError(err, resourceBoard)
	}

	if rowsAffected == 0 {
		return domain.ErrForbidden
	}

	return nil
}
