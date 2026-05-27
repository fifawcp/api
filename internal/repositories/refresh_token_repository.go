package repositories

import (
	"context"
	"database/sql"
	"time"

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
	// session.expires_at to the service for capping the rotated token's TTL.
	// rotated_at is carried too: a rotated-but-unexpired row is still returned so
	// the service can grant it the grace window (or reject it as stale reuse).
	query := `SELECT
		rt.id,
		rt.user_id,
		rt.session_id,
		rt.token_hash,
		rt.expires_at,
		rt.rotated_at,
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
		&refreshToken.RotatedAt,
		&refreshToken.CreatedAt,
		&refreshToken.SessionExpiresAt,
	)

	if err != nil {
		return nil, handleDBError(err, resourceRefreshToken)
	}

	return &refreshToken, nil
}

// RotateRefreshToken supersedes the presented token and issues the new one in a
// single statement:
//  1. mark the old token rotated (idempotent — a concurrent caller may have
//     already marked it; we still issue the new token so the race loser keeps a
//     working session),
//  2. prune the session's *previous* grace token (any rotated row other than the
//     one we just superseded), bounding an active session to ≤2 rows,
//  3. insert the new token.
//
// The caller is responsible for validating the old token (existence, expiry,
// grace window) before calling this.
func (r *RefreshTokenRepository) RotateRefreshToken(
	ctx context.Context,
	oldTokenHash string,
	newToken *domain.RefreshToken,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `WITH rotated AS (
		UPDATE refresh_tokens
		SET rotated_at = NOW()
		WHERE token_hash = $1 AND rotated_at IS NULL AND expires_at > NOW()
		RETURNING id
	),
	pruned AS (
		DELETE FROM refresh_tokens
		WHERE session_id = $3 AND rotated_at IS NOT NULL AND token_hash <> $1
		RETURNING id
	)
	INSERT INTO refresh_tokens (
		user_id,
		session_id,
		token_hash,
		expires_at
	)
	VALUES ($2, $3, $4, $5)
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

	if err != nil {
		return handleDBError(err, resourceRefreshToken)
	}

	return nil
}

// DeleteRotatedBefore removes superseded tokens whose grace window has elapsed.
// Active sessions are already bounded by the prune in RotateRefreshToken; this
// sweeps the stragglers left by sessions that rotated once and then went idle.
func (r *RefreshTokenRepository) DeleteRotatedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	res, err := r.db.ExecContext(
		ctx,
		`DELETE FROM refresh_tokens WHERE rotated_at IS NOT NULL AND rotated_at < $1`,
		cutoff,
	)
	if err != nil {
		return 0, handleDBError(err, resourceRefreshToken)
	}

	return res.RowsAffected()
}
