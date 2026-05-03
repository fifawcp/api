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

func (r *BoardRepository) GetUserBoards(ctx context.Context, userID string) ([]*domain.UserBoardListItem, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT b.id, b.name
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
		if err := rows.Scan(&board.ID, &board.Name); err != nil {
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
	boardID string,
	userID string,
) (*domain.BoardDetails, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH ranked AS (
			SELECT
				bm.user_id,
				bm.created_at AS joined_at,
				RANK() OVER (
					ORDER BY us.total_points DESC, us.updated_at ASC, bm.user_id ASC
				) AS rank,
				COALESCE(us.total_points, 0) AS total_points
			FROM board_members bm
			LEFT JOIN user_scores us ON us.user_id = bm.user_id
			WHERE bm.board_id = $1
		)
		SELECT
			b.id,
			b.name,
			b.owner_user_id,
			b.join_code,
			b.privacy,
			b.created_at,
			r.joined_at,
			r.rank,
			r.total_points
		FROM boards b
		LEFT JOIN ranked r ON r.user_id = $2
		WHERE b.id = $1
	`

	var details domain.BoardDetails
	var ownerUserID, joinCode sql.NullString
	var joinedAt sql.NullTime
	var userRank, totalPoints sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, boardID, userID).Scan(
		&details.ID,
		&details.Name,
		&ownerUserID,
		&joinCode,
		&details.Privacy,
		&details.CreatedAt,
		&joinedAt,
		&userRank,
		&totalPoints,
	)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	if ownerUserID.Valid {
		details.OwnerUserID = &ownerUserID.String
	}

	if joinCode.Valid {
		details.JoinCode = &joinCode.String
	}

	if joinedAt.Valid {
		details.JoinedAt = joinedAt.Time
	}

	if userRank.Valid {
		details.UserRank = int(userRank.Int64)
	}

	if totalPoints.Valid {
		details.UserTotalPoints = int(totalPoints.Int64)
	}

	return &details, nil
}

func (r *BoardRepository) GetBoardMembers(
	ctx context.Context,
	boardID string,
	page, limit int,
) (*domain.BoardMembersPage, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	offset := (page - 1) * limit

	// RANK() is computed over the full board, then OFFSET/LIMIT slices the page
	// COUNT(*) OVER () returns the full member count alongside each row
	query := `
		WITH ranked AS (
			SELECT
				bm.user_id,
				u.username,
				u.first_name,
				u.last_name,
				bm.role,
				bm.created_at AS joined_at,
				us.total_points,
				us.pickem_points,
				us.match_score_points,
				us.exact_hits,
				us.correct_outcomes,
				us.updated_at,
				RANK() OVER (
					ORDER BY us.total_points DESC, us.updated_at ASC, bm.user_id ASC
				) AS rank,
				COUNT(*) OVER () AS total
			FROM board_members bm
			INNER JOIN users u        ON u.id       = bm.user_id
			LEFT  JOIN user_scores us ON us.user_id = bm.user_id
			WHERE bm.board_id = $1
		)
		SELECT
			user_id,
			username,
			first_name,
			last_name,
			role,
			joined_at,
			rank,
			total_points,
			pickem_points,
			match_score_points,
			exact_hits,
			correct_outcomes,
			updated_at,
			total
		FROM ranked
		ORDER BY rank ASC, joined_at ASC
		OFFSET $2 LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, boardID, offset, limit)
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
			&member.Rank,
			&member.TotalPoints,
			&member.PickemPoints,
			&member.MatchScorePoints,
			&member.ExactHits,
			&member.CorrectOutcomes,
			&member.UpdatedAt,
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

	// If the requested page is past the end, the windowed COUNT didn't return
	// Fall back to a direct count so pagination.total is still populated
	if len(membersPage.Members) == 0 {
		countQuery := `SELECT COUNT(*) FROM board_members WHERE board_id = $1`
		if err := r.db.QueryRowContext(ctx, countQuery, boardID).Scan(&membersPage.Pagination.Total); err != nil {
			return nil, handleDBError(err, resourceBoard)
		}
	}

	membersPage.Pagination.HasMore = page*limit < membersPage.Pagination.Total

	return membersPage, nil
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
			privacy,
			created_at
		FROM boards
		WHERE id = $1
	`

	var board domain.Board
	var ownerUserID, joinCode sql.NullString

	err := r.db.QueryRowContext(ctx, query, boardID).Scan(
		&board.ID,
		&board.Name,
		&ownerUserID,
		&joinCode,
		&board.Privacy,
		&board.CreatedAt,
	)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	if ownerUserID.Valid {
		board.OwnerUserID = &ownerUserID.String
	}

	if joinCode.Valid {
		board.JoinCode = &joinCode.String
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
