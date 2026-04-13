package domain

import (
	"context"
	"encoding/json"
	"time"
)

type Session struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id"`
	DeviceInfo json.RawMessage `json:"device_info"`
	IPAddress  string          `json:"ip_address"`
	UserAgent  string          `json:"user_agent"`
	LastUsedAt time.Time       `json:"last_used_at"`
	ExpiresAt  time.Time       `json:"expires_at"`
	CreatedAt  time.Time       `json:"created_at"`
}

type SessionRepositoryInterface interface {
	CreateSession(ctx context.Context, session *Session) error
	GetSessions(ctx context.Context, refreshTokenHash string) ([]Session, error)
	UpdateLastUsedAt(ctx context.Context, id string) error
	DeleteSession(ctx context.Context, refreshTokenHash string) error
	DeleteAllSessions(ctx context.Context, refreshTokenHash string) error
	DeleteSessionById(ctx context.Context, sessionID string) error
}
