package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
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

func (r *BoardRepository) CreateBoard(
	ctx context.Context,
	board *domain.Board,
	ownerID string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourceBoard)
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx,
		`INSERT INTO boards (name, join_code)
		VALUES ($1, $2)
		RETURNING id, created_at`,
		board.Name, board.JoinCode,
	).Scan(&board.ID, &board.CreatedAt); err != nil {
		return handleDBError(err, resourceBoard)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO board_members (board_id, user_id, role)
		VALUES ($1, $2, 'owner')`,
		board.ID, ownerID,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	return tx.Commit()
}

func (r *BoardRepository) GetUserBoards(ctx context.Context, userID string) ([]*domain.UserBoardListItem, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT b.id, b.name, b.privacy, bm.role
		FROM boards b
		INNER JOIN board_members bm ON bm.board_id = b.id
		WHERE bm.user_id = $1
		ORDER BY b.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}
	defer rows.Close()

	boards := []*domain.UserBoardListItem{}
	for rows.Next() {
		var board domain.UserBoardListItem
		if err := rows.Scan(&board.ID, &board.Name, &board.Privacy, &board.Role); err != nil {
			return nil, handleDBError(err, resourceBoard)
		}

		boards = append(boards, &board)
	}

	if err = rows.Err(); err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	return boards, nil
}

func (r *BoardRepository) GetBoardDetails(
	ctx context.Context,
	boardID int64,
	userID string,
) (*domain.BoardDetails, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		b.id,
		b.name,
		b.join_code,
		b.privacy,
		b.created_at,
		(SELECT COUNT(*) FROM board_members WHERE board_id = b.id)  AS member_count,
		(SELECT COUNT(*) FROM competitions WHERE board_id = b.id)   AS competition_count,
		bm.role,
		bm.created_at AS joined_at
	FROM boards b
	LEFT JOIN board_members bm ON bm.board_id = b.id AND bm.user_id = $2
	WHERE b.id = $1`

	var details domain.BoardDetails
	var joinCode sql.NullString
	var role sql.NullString
	var joinedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, boardID, userID).Scan(
		&details.ID,
		&details.Name,
		&joinCode,
		&details.Privacy,
		&details.CreatedAt,
		&details.MemberCount,
		&details.CompetitionCount,
		&role,
		&joinedAt,
	)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	if joinCode.Valid {
		details.JoinCode = &joinCode.String
	}

	if role.Valid {
		details.Viewer.Role = domain.BoardMemberRole(role.String)
	}

	if joinedAt.Valid {
		details.Viewer.JoinedAt = joinedAt.Time
	}

	return &details, nil
}

func (r *BoardRepository) GetBoardMembers(
	ctx context.Context,
	boardID int64,
	filters domain.BoardMembersFilters,
	page, limit int,
) (*domain.BoardMembersPage, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	offset := (page - 1) * limit

	args := []any{boardID}
	searchClause := ""

	if filters.Search != "" {
		args = append(args, "%"+filters.Search+"%")
		searchClause = `AND (
				u.first_name ILIKE $2
				OR u.last_name ILIKE $2
				OR u.username ILIKE $2
			)`
	}

	offsetPlaceholder := "$" + strconv.Itoa(len(args)+1)
	limitPlaceholder := "$" + strconv.Itoa(len(args)+2)
	args = append(args, offset, limit)

	query := `
		SELECT
			bm.user_id,
			u.username,
			u.first_name,
			u.last_name,
			bm.role,
			bm.created_at AS joined_at,
			COUNT(*) OVER () AS total
		FROM board_members bm
		INNER JOIN users u ON u.id = bm.user_id
		WHERE bm.board_id = $1
			` + searchClause + `
		ORDER BY bm.created_at DESC, bm.user_id ASC
		OFFSET ` + offsetPlaceholder + ` LIMIT ` + limitPlaceholder

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}
	defer rows.Close()

	membersPage := &domain.BoardMembersPage{
		Members: []*domain.BoardMemberDetails{},
		Pagination: domain.Pagination{
			Page:  page,
			Limit: limit,
		},
	}

	for rows.Next() {
		var member domain.BoardMemberDetails
		var membersTotal int

		if err := rows.Scan(
			&member.UserID,
			&member.UserName,
			&member.FirstName,
			&member.LastName,
			&member.Role,
			&member.JoinedAt,
			&membersTotal,
		); err != nil {
			return nil, handleDBError(err, resourceBoard)
		}

		membersPage.Pagination.Total = membersTotal
		membersPage.Members = append(membersPage.Members, &member)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	if len(membersPage.Members) == 0 {
		countArgs := []any{boardID}
		countQuery := `
			SELECT COUNT(*)
			FROM board_members bm
			INNER JOIN users u ON u.id = bm.user_id
			WHERE bm.board_id = $1`

		if filters.Search != "" {
			countArgs = append(countArgs, "%"+filters.Search+"%")
			countQuery += ` AND (u.first_name ILIKE $2 OR u.last_name ILIKE $2 OR u.username ILIKE $2)`
		}

		if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&membersPage.Pagination.Total); err != nil {
			return nil, handleDBError(err, resourceBoard)
		}
	}

	membersPage.Pagination.HasMore = page*limit < membersPage.Pagination.Total

	return membersPage, nil
}

func (r *BoardRepository) GetBoardPreview(
	ctx context.Context,
	joinCode string,
	sampleSize int,
) (*domain.BoardPreview, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	var preview domain.BoardPreview
	var boardID int64

	err := r.db.QueryRowContext(ctx, `
		SELECT
			b.id,
			b.name,
			b.privacy,
			(SELECT COUNT(*) FROM board_members WHERE board_id = b.id) AS member_count
		FROM boards b
		WHERE b.join_code = $1`,
		joinCode,
	).Scan(&boardID, &preview.Name, &preview.Privacy, &preview.MemberCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrBoardNotFound
		}
		return nil, handleDBError(err, resourceBoard)
	}

	preview.Members = []*domain.BoardPreviewMember{}

	rows, err := r.db.QueryContext(ctx, `
		SELECT bm.user_id, u.username, u.first_name, u.last_name
		FROM board_members bm
		INNER JOIN users u ON u.id = bm.user_id
		WHERE bm.board_id = $1
		ORDER BY (bm.role = 'owner') DESC, bm.created_at ASC, bm.user_id ASC
		LIMIT $2`,
		boardID, sampleSize,
	)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}
	defer rows.Close()

	for rows.Next() {
		var member domain.BoardPreviewMember
		if err := rows.Scan(&member.UserID, &member.UserName, &member.FirstName, &member.LastName); err != nil {
			return nil, handleDBError(err, resourceBoard)
		}
		preview.Members = append(preview.Members, &member)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	return &preview, nil
}

func (r *BoardRepository) GetBoardByID(ctx context.Context, boardID int64) (*domain.Board, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT id, name, join_code, privacy, created_at
		FROM boards
		WHERE id = $1
	`

	var board domain.Board
	var joinCode sql.NullString

	err := r.db.QueryRowContext(ctx, query, boardID).Scan(
		&board.ID,
		&board.Name,
		&joinCode,
		&board.Privacy,
		&board.CreatedAt,
	)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	if joinCode.Valid {
		board.JoinCode = &joinCode.String
	}

	return &board, nil
}

func (r *BoardRepository) UpdateJoinCode(ctx context.Context, boardID int64, joinCode string) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `UPDATE boards SET join_code = $1 WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, joinCode, boardID)
	if err != nil {
		return handleDBError(err, resourceBoard)
	}

	return nil
}

func (r *BoardRepository) UpdateBoard(ctx context.Context, boardID int64, board *domain.Board) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

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
	boardID int64,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `DELETE FROM boards WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, boardID)
	if err != nil {
		return handleDBError(err, resourceBoard)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleDBError(err, resourceBoard)
	}

	if rowsAffected == 0 {
		return domain.ErrBoardNotFound
	}

	return nil
}
