package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
)

type UserRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewUserRepository(db *sql.DB, cfg *config.Config) *UserRepository {
	return &UserRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *UserRepository) CreateUser(
	ctx context.Context,
	user *domain.User,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	userQuery := `INSERT INTO users
		(
			first_name,
			last_name,
			username,
			email
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	if err := tx.QueryRowContext(
		ctx,
		userQuery,
		user.FirstName,
		user.LastName,
		user.Username,
		user.Email,
	).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return handleDBError(err, resourceUser)
	}

	var globalBoardID int64
	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM boards WHERE privacy = 'global'`,
	).Scan(&globalBoardID); err != nil {
		return handleDBError(err, resourceBoard)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO board_members (board_id, user_id, role) VALUES ($1, $2, 'member')`,
		globalBoardID,
		user.ID,
	); err != nil {
		return handleDBError(err, resourceBoardMember)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO competition_match_scores (competition_id, user_id)
		 SELECT id, $1 FROM competitions WHERE board_id = $2 AND type = 'match'`,
		user.ID,
		globalBoardID,
	); err != nil {
		return handleDBError(err, resourceCompetitionScore)
	}

	return tx.Commit()
}

func (r *UserRepository) GetUserByIdentifier(
	ctx context.Context,
	identifier string,
) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		id,
		first_name,
		last_name,
		username,
		email,
		role,
		created_at,
		updated_at
	FROM users
	WHERE LOWER(email) = LOWER($1) OR username = $1`

	var user domain.User

	err := r.db.QueryRowContext(ctx, query, identifier).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, handleDBError(err, resourceUser)
	}

	return &user, nil
}

func (r *UserRepository) GetUserByID(
	ctx context.Context,
	userID string,
) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		id,
		first_name,
		last_name,
		username,
		email,
		role,
		created_at,
		updated_at
	FROM users
	WHERE id = $1`

	var user domain.User

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, handleDBError(err, resourceUser)
	}

	return &user, nil
}

func (r *UserRepository) UpdateUser(
	ctx context.Context,
	userID string,
	updates domain.UserUpdate,
) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	setClauses := []string{}
	args := []any{}
	argIndex := 1

	if updates.FirstName != nil {
		setClauses = append(setClauses, fmt.Sprintf("first_name = $%d", argIndex))
		args = append(args, *updates.FirstName)
		argIndex++
	}
	if updates.LastName != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_name = $%d", argIndex))
		args = append(args, *updates.LastName)
		argIndex++
	}
	if updates.Username != nil {
		setClauses = append(setClauses, fmt.Sprintf("username = $%d", argIndex))
		args = append(args, *updates.Username)
		argIndex++
	}

	if len(setClauses) == 0 {
		return r.GetUserByID(ctx, userID)
	}

	// No BEFORE UPDATE trigger exists on users, so bump updated_at explicitly.
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, userID)

	query := fmt.Sprintf(`UPDATE users
		SET %s
		WHERE id = $%d
		RETURNING id, first_name, last_name, username, email, role, created_at, updated_at`,
		strings.Join(setClauses, ", "),
		argIndex,
	)

	var user domain.User

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, handleDBError(err, resourceUser)
	}

	return &user, nil
}
