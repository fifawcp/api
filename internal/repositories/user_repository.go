package repositories

import (
	"context"
	"database/sql"

	"github.com/ncondes/fifa-world-cup-pickems/internal/domain"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/config"
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

	query := `INSERT INTO users
		(
			first_name,
			last_name,
			username,
			email
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(
		ctx,
		query,
		user.FirstName,
		user.LastName,
		user.Username,
		user.Email,
	).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return handleDBError(err, resourceUser)
	}

	return nil
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
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, handleDBError(err, resourceUser)
	}

	return &user, nil
}
