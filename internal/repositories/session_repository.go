package repositories

import (
	"context"
	"database/sql"

	"github.com/ncondes/fifa-world-cup-pickems/internal/domain"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/config"
)

type SessionRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewSessionRepository(
	db *sql.DB,
	cfg *config.Config,
) *SessionRepository {
	return &SessionRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *SessionRepository) CreateSession(
	ctx context.Context,
	session *domain.Session,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `INSERT INTO sessions (
		user_id,
		device_info,
		ip_address,
		user_agent,
		expires_at
	) VALUES ($1, $2, $3, $4, $5)
	 RETURNING id`

	err := r.db.QueryRowContext(
		ctx,
		query,
		session.UserID,
		session.DeviceInfo,
		session.IPAddress,
		session.UserAgent,
		session.ExpiresAt,
	).Scan(&session.ID)

	if err != nil {
		return handleDBError(err, resourceSession)
	}

	return nil
}

func (r *SessionRepository) GetSessions(
	ctx context.Context,
	refreshTokenHash string,
) ([]domain.Session, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		s.id,
		s.user_id,
		s.device_info,
		s.ip_address,
		s.user_agent,
		s.last_used_at,
		s.expires_at,
		s.created_at
	FROM refresh_tokens rt
	JOIN sessions s ON s.user_id = rt.user_id
	WHERE rt.token_hash = $1 AND rt.expires_at > NOW()`

	rows, err := r.db.QueryContext(ctx, query, refreshTokenHash)
	if err != nil {
		return nil, handleDBError(err, resourceSession)
	}
	defer rows.Close()

	var sessions []domain.Session

	for rows.Next() {
		var session domain.Session

		if err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.DeviceInfo,
			&session.IPAddress,
			&session.UserAgent,
			&session.LastUsedAt,
			&session.ExpiresAt,
			&session.CreatedAt,
		); err != nil {
			return nil, handleDBError(err, resourceSession)
		}

		sessions = append(sessions, session)
	}

	if len(sessions) == 0 {
		return nil, domain.ErrRefreshTokenInvalidOrExpired
	}

	if err = rows.Err(); err != nil {
		return nil, handleDBError(err, resourceSession)
	}

	return sessions, nil
}

func (r *SessionRepository) UpdateLastUsedAt(
	ctx context.Context,
	id string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `UPDATE sessions SET last_used_at = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return handleDBError(err, resourceSession)
	}

	// Check if session exists
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleDBError(err, resourceSession)
	}

	if rowsAffected == 0 {
		return domain.ErrSessionNotFound
	}

	return nil
}

func (r *SessionRepository) DeleteSession(
	ctx context.Context,
	refreshTokenHash string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `DELETE FROM sessions
	WHERE id = (
		SELECT session_id FROM refresh_tokens
		WHERE token_hash = $1 AND expires_at > NOW()
	)`

	result, err := r.db.ExecContext(ctx, query, refreshTokenHash)
	if err != nil {
		return handleDBError(err, resourceSession)
	}

	// Check if session exists
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleDBError(err, resourceSession)
	}

	if rowsAffected == 0 {
		return domain.ErrRefreshTokenInvalidOrExpired
	}

	return nil
}

func (r *SessionRepository) DeleteAllSessions(
	ctx context.Context,
	refreshTokenHash string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `DELETE FROM sessions
	WHERE user_id = (
		SELECT user_id FROM refresh_tokens
		WHERE token_hash = $1 AND expires_at > NOW()
	)`

	result, err := r.db.ExecContext(ctx, query, refreshTokenHash)
	if err != nil {
		return handleDBError(err, resourceSession)
	}

	// Check if any sessions were deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleDBError(err, resourceSession)
	}

	if rowsAffected == 0 {
		return domain.ErrRefreshTokenInvalidOrExpired
	}

	return nil
}

func (r *SessionRepository) DeleteSessionById(
	ctx context.Context,
	sessionID string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `DELETE FROM sessions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return handleDBError(err, resourceSession)
	}

	// Check if session exists
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleDBError(err, resourceSession)
	}

	if rowsAffected == 0 {
		return domain.ErrSessionNotFound
	}

	return nil
}
