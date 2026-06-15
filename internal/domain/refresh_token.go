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

// IssuedTokens is the access/refresh pair a refresh hands back, cached briefly so
// concurrent refreshes of the same token converge on one successor.
type IssuedTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RefreshReplayStorage deduplicates concurrent refreshes of the same token.
type RefreshReplayStorage interface {
	// Claim reserves oldTokenHash for the caller. won=true means no successor was cached yet
	// (caller should rotate); won=false returns the successor a concurrent refresh issued.
	Claim(ctx context.Context, oldTokenHash string, tokens *IssuedTokens, ttl time.Duration) (won bool, existing *IssuedTokens, err error)
	// Release drops a claim so a failed rotation doesn't strand losers on an unpersisted successor.
	Release(ctx context.Context, oldTokenHash string) error
}
