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

func (r *BoardRepository) GetUserBoards(ctx context.Context, userID string) ([]*domain.BoardSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	// Rank only boards the user belongs to, then select the user's row from each board.
	query := `
		WITH user_boards AS (
			SELECT board_id, created_at AS joined_at
			FROM board_members
			WHERE user_id = $1
		),
		board_member_counts AS (
			SELECT board_id, COUNT(user_id) AS members_count
			FROM board_members
			WHERE board_id IN (SELECT board_id FROM user_boards)
			GROUP BY board_id
		),
		ranked AS (
			SELECT
				board_id,
				user_id,
				RANK() OVER (
					PARTITION BY board_id
					ORDER BY total_points DESC, updated_at ASC, user_id ASC
				) AS rank
			FROM board_rankings
			WHERE board_id IN (SELECT board_id FROM user_boards)
		)
		SELECT
			b.id,
			b.name,
			b.owner_user_id,
			b.created_at,
			ub.joined_at,
			r.rank AS user_rank,
			bmc.members_count
		FROM boards b
		INNER JOIN user_boards ub
			ON ub.board_id = b.id
		INNER JOIN board_member_counts bmc
			ON bmc.board_id = b.id
		LEFT JOIN ranked r
			ON r.board_id = b.id AND r.user_id = $1
		ORDER BY b.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}
	defer rows.Close()

	boards := []*domain.BoardSummary{}

	for rows.Next() {
		var board domain.BoardSummary
		err := rows.Scan(
			&board.ID,
			&board.Name,
			&board.OwnerUserID,
			&board.CreatedAt,
			&board.JoinedAt,
			&board.UserRank,
			&board.MembersCount,
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

func (r *BoardRepository) GetBoardDetails(ctx context.Context, boardID string) (*domain.BoardDetails, error) {
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

	var details domain.BoardDetails

	err := r.db.QueryRowContext(ctx, query, boardID).Scan(
		&details.ID,
		&details.Name,
		&details.OwnerUserID,
		&details.JoinCode,
		&details.CreatedAt,
	)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	// Board fields are loaded once, then members are attached from the ranked query below
	membersQuery := `
		WITH ranked AS (
			SELECT
				board_id,
				user_id,
				total_points,
				global_points,
				detailed_points,
				exact_hits,
				correct_outcomes,
				updated_at,
				RANK() OVER (
					PARTITION BY board_id
					ORDER BY total_points DESC, updated_at ASC, user_id ASC
				) AS rank
			FROM board_rankings
			WHERE board_id = $1
		)
		SELECT
			bm.user_id,
			u.username,
			bm.role,
			bm.created_at AS joined_at,
			r.rank AS rank,
			r.total_points,
			r.global_points,
			r.detailed_points,
			r.exact_hits,
			r.correct_outcomes,
			r.updated_at
		FROM board_members bm
		INNER JOIN users u ON bm.user_id = u.id
		LEFT JOIN ranked r ON r.board_id = bm.board_id AND r.user_id = bm.user_id
		WHERE bm.board_id = $1
		ORDER BY rank ASC, bm.created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, membersQuery, boardID)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}
	defer rows.Close()

	details.Members = []*domain.BoardDetailsMember{}

	for rows.Next() {
		var member domain.BoardDetailsMember
		if err := rows.Scan(
			&member.UserID,
			&member.UserName,
			&member.Role,
			&member.JoinedAt,
			&member.Rank,
			&member.TotalPoints,
			&member.GlobalPoints,
			&member.DetailedPoints,
			&member.ExactHits,
			&member.CorrectOutcomes,
			&member.UpdatedAt,
		); err != nil {
			return nil, handleDBError(err, resourceBoard)
		}

		details.Members = append(details.Members, &member)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	return &details, nil
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
