package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockRefreshTokenRepository struct {
	CreateRefreshTokenFunc         func(ctx context.Context, refreshToken *domain.RefreshToken) error
	GetRefreshTokenByTokenHashFunc func(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	RotateRefreshTokenFunc         func(ctx context.Context, oldTokenHash string, newToken *domain.RefreshToken) error
}

func (m *MockRefreshTokenRepository) CreateRefreshToken(ctx context.Context, refreshToken *domain.RefreshToken) error {
	if m.CreateRefreshTokenFunc != nil {
		return m.CreateRefreshTokenFunc(ctx, refreshToken)
	}
	panic("CreateRefreshToken called unexpectedly")
}

func (m *MockRefreshTokenRepository) GetRefreshTokenByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	if m.GetRefreshTokenByTokenHashFunc != nil {
		return m.GetRefreshTokenByTokenHashFunc(ctx, tokenHash)
	}
	panic("GetRefreshTokenByTokenHash called unexpectedly")
}

func (m *MockRefreshTokenRepository) RotateRefreshToken(ctx context.Context, oldTokenHash string, newToken *domain.RefreshToken) error {
	if m.RotateRefreshTokenFunc != nil {
		return m.RotateRefreshTokenFunc(ctx, oldTokenHash, newToken)
	}
	panic("RotateRefreshToken called unexpectedly")
}
