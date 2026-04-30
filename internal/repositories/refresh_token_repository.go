package repositories

import (
	"context"
	"database/sql"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
)

type RefreshTokenRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewRefreshTokenRepository(
	db *sql.DB,
	cfg *config.Config,
) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *RefreshTokenRepository) CreateRefreshToken(
	ctx context.Context,
	refreshToken *domain.RefreshToken,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `INSERT INTO refresh_tokens (
		user_id,
		session_id,
		token_hash,
		expires_at
	) VALUES ($1, $2, $3, $4)`

	_, err := r.db.ExecContext(
		ctx,
		query,
		refreshToken.UserID,
		refreshToken.SessionID,
		refreshToken.TokenHash,
		refreshToken.ExpiresAt,
	)

	if err != nil {
		return handleDBError(err, resourceRefreshToken)
	}

	return nil
}

func (r *RefreshTokenRepository) GetRefreshTokenByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*domain.RefreshToken, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	// Join sessions so we can enforce session expiry at lookup time and carry
	// session.expires_at to the service for capping the rotated token's TTL
	query := `SELECT
		rt.id,
		rt.user_id,
		rt.session_id,
		rt.token_hash,
		rt.expires_at,
		rt.created_at,
		s.expires_at AS session_expires_at
	FROM refresh_tokens rt
	JOIN sessions s ON s.id = rt.session_id
	WHERE rt.token_hash = $1
	  AND rt.expires_at > NOW()
	  AND s.expires_at > NOW()`

	var refreshToken domain.RefreshToken

	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&refreshToken.ID,
		&refreshToken.UserID,
		&refreshToken.SessionID,
		&refreshToken.TokenHash,
		&refreshToken.ExpiresAt,
		&refreshToken.CreatedAt,
		&refreshToken.SessionExpiresAt,
	)

	if err != nil {
		return nil, handleDBError(err, resourceRefreshToken)
	}

	return &refreshToken, nil
}

func (r *RefreshTokenRepository) RotateRefreshToken(
	ctx context.Context,
	oldTokenHash string,
	newToken *domain.RefreshToken,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `WITH deleted AS (
		DELETE FROM refresh_tokens
		WHERE token_hash = $1 AND expires_at > NOW()
		RETURNING id
	)
	INSERT INTO refresh_tokens (
		user_id,
		session_id,
		token_hash,
		expires_at
	)
	SELECT $2, $3, $4, $5
	WHERE EXISTS (SELECT 1 FROM deleted)
	RETURNING id`

	err := r.db.QueryRowContext(
		ctx,
		query,
		oldTokenHash,
		newToken.UserID,
		newToken.SessionID,
		newToken.TokenHash,
		newToken.ExpiresAt,
	).Scan(&newToken.ID)

	if err == sql.ErrNoRows {
		return handleDBError(err, resourceRefreshToken)
	}

	if err != nil {
		return handleDBError(err, resourceRefreshToken)
	}

	return nil
}
