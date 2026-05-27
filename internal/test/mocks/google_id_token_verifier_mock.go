package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockGoogleIDTokenVerifier struct {
	VerifyFunc func(ctx context.Context, rawIDToken string) (*domain.IDToken, error)
}

func (m *MockGoogleIDTokenVerifier) Verify(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(ctx, rawIDToken)
	}
	panic("Verify called unexpectedly")
}
