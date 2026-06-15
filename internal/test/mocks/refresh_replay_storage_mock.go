package mocks

import (
	"context"
	"time"

	"github.com/fifawcp/api/internal/domain"
)

type MockRefreshReplayStorage struct {
	ClaimFunc   func(ctx context.Context, oldTokenHash string, tokens *domain.IssuedTokens, ttl time.Duration) (bool, *domain.IssuedTokens, error)
	ReleaseFunc func(ctx context.Context, oldTokenHash string) error
}

func (m *MockRefreshReplayStorage) Claim(ctx context.Context, oldTokenHash string, tokens *domain.IssuedTokens, ttl time.Duration) (bool, *domain.IssuedTokens, error) {
	if m.ClaimFunc != nil {
		return m.ClaimFunc(ctx, oldTokenHash, tokens, ttl)
	}
	panic("Claim called unexpectedly")
}

func (m *MockRefreshReplayStorage) Release(ctx context.Context, oldTokenHash string) error {
	if m.ReleaseFunc != nil {
		return m.ReleaseFunc(ctx, oldTokenHash)
	}
	panic("Release called unexpectedly")
}
