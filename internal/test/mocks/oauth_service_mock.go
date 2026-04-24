package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/dtos"
)

type MockOAuthService struct {
	BeginGoogleLoginFunc    func(ctx context.Context, returnTo string) (string, error)
	CompleteGoogleLoginFunc func(ctx context.Context, state string, code string, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, string, error)
}

func (m *MockOAuthService) BeginGoogleLogin(ctx context.Context, returnTo string) (string, error) {
	if m.BeginGoogleLoginFunc != nil {
		return m.BeginGoogleLoginFunc(ctx, returnTo)
	}
	panic("BeginGoogleLogin called unexpectedly")
}

func (m *MockOAuthService) CompleteGoogleLogin(ctx context.Context, state string, code string, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, string, error) {
	if m.CompleteGoogleLoginFunc != nil {
		return m.CompleteGoogleLoginFunc(ctx, state, code, requestInfo)
	}
	panic("CompleteGoogleLogin called unexpectedly")
}
