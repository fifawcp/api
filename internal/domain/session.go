package domain

import (
	"context"
	"encoding/json"
	"time"
)

type Session struct {
	ID         string          `json:"id" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	UserID     string          `json:"user_id" example:"3f01faf6-d1a5-494a-bd41-b0475b6c8d03"`
	DeviceInfo json.RawMessage `json:"device_info" swaggertype:"object"`
	IPAddress  string          `json:"ip_address" example:"192.168.1.1"`
	UserAgent  string          `json:"user_agent" example:"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"`
	LastUsedAt time.Time       `json:"last_used_at" example:"2026-04-13T01:30:00Z"`
	ExpiresAt  time.Time       `json:"expires_at" example:"2026-05-13T01:30:00Z"`
	CreatedAt  time.Time       `json:"created_at" example:"2026-04-12T18:47:00Z"`
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session *Session) error
	GetSessions(ctx context.Context, refreshTokenHash string) ([]Session, error)
	UpdateLastUsedAt(ctx context.Context, id string, slideTo time.Time, maxLifetime time.Duration) (time.Time, error)
	DeleteSession(ctx context.Context, refreshTokenHash string) error
	DeleteAllSessions(ctx context.Context, refreshTokenHash string) error
	DeleteSessionById(ctx context.Context, sessionID string, userID string) error
	DeleteExpiredSessions(ctx context.Context) (int64, error)
}
