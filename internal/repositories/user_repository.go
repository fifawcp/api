package repositories

import (
	"context"
	"database/sql"

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

	if _, err := tx.ExecContext(ctx, `INSERT INTO user_scores (user_id) VALUES ($1)`, user.ID); err != nil {
		return handleDBError(err, resourceUserScore)
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
