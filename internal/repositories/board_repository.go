package repositories

import (
	"context"
	"database/sql"
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
		SELECT b.id, b.name, b.privacy
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
		if err := rows.Scan(&board.ID, &board.Name, &board.Privacy); err != nil {
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
				bm.role,
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
			r.role,
			r.joined_at,
			r.rank,
			r.total_points
		FROM boards b
		LEFT JOIN ranked r ON r.user_id = $2
		WHERE b.id = $1
	`

	var details domain.BoardDetails
	var ownerUserID, joinCode sql.NullString
	var role sql.NullString
	var joinedAt sql.NullTime
	var rank, totalPoints sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, boardID, userID).Scan(
		&details.ID,
		&details.Name,
		&ownerUserID,
		&joinCode,
		&details.Privacy,
		&details.CreatedAt,
		&role,
		&joinedAt,
		&rank,
		&totalPoints,
	)
	if err != nil {
		return nil, handleDBError(err, resourceBoard)
	}

	if ownerUserID.Valid {
		details.OwnerUserID = &ownerUserID.String
		details.Viewer.IsOwner = ownerUserID.String == userID
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

	if rank.Valid {
		details.Viewer.Rank = int(rank.Int64)
	}

	if totalPoints.Valid {
		details.Viewer.TotalPoints = int(totalPoints.Int64)
	}

	return &details, nil
}

func (r *BoardRepository) GetBoardMembers(
	ctx context.Context,
	boardID string,
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

	// RANK() is computed over the full filtered board, then OFFSET/LIMIT slices the page
	// RANK is always by total_points DESC regardless of the user's chosen sort, so it
	// reflects the leaderboard position rather than the row's position in the current view
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
				` + searchClause + `
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
		ORDER BY ` + boardMembersSortColumn(filters.Sort) + ` DESC NULLS LAST, joined_at ASC, user_id ASC
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

	// If the requested page is past the end (or the search filter excluded everyone),
	// the windowed COUNT never reached us. Fall back to a direct count that respects
	// the same WHERE so pagination.total stays accurate.
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

func boardMembersSortColumn(sort domain.BoardMembersSort) string {
	switch sort {
	case domain.BoardMembersSortPickemPoints:
		return "pickem_points"
	case domain.BoardMembersSortMatchScorePoints:
		return "match_score_points"
	case domain.BoardMembersSortExactHits:
		return "exact_hits"
	case domain.BoardMembersSortCorrectOutcomes:
		return "correct_outcomes"
	default:
		return "total_points"
	}
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
