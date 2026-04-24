package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockGoogleOAuth2Client struct {
	BuildAuthCodeURLFunc     func(state string) string
	ExchangeCodeForTokenFunc func(ctx context.Context, code string) (*domain.OIDCToken, error)
}

func (m *MockGoogleOAuth2Client) BuildAuthCodeURL(state string) string {
	if m.BuildAuthCodeURLFunc != nil {
		return m.BuildAuthCodeURLFunc(state)
	}
	panic("BuildAuthCodeURL called unexpectedly")
}

func (m *MockGoogleOAuth2Client) ExchangeCodeForToken(ctx context.Context, code string) (*domain.OIDCToken, error) {
	if m.ExchangeCodeForTokenFunc != nil {
		return m.ExchangeCodeForTokenFunc(ctx, code)
	}
	panic("ExchangeCodeForToken called unexpectedly")
}
