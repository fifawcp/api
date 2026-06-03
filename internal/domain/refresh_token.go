package domain

import (
	"context"
	"time"
)

type RefreshToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	SessionID string     `json:"session_id"`
	TokenHash string     `json:"token_hash"`
	ExpiresAt time.Time  `json:"expires_at"`
	RotatedAt *time.Time `json:"rotated_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type RefreshTokenRepository interface {
	CreateRefreshToken(ctx context.Context, refreshToken *RefreshToken) error
	GetRefreshTokenByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RotateRefreshToken(ctx context.Context, oldTokenHash string, newToken *RefreshToken) error
	DeleteRotatedBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
