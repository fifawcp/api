package testutils

import (
	"context"

	"github.com/ncondes/fifawcp/internal/domain"
	"github.com/ncondes/fifawcp/internal/dtos"
)

type MockAuthService struct {
	RequestOtpFunc    func(ctx context.Context, payload *dtos.RequestOtpDto) error
	AuthenticateFunc  func(ctx context.Context, payload *dtos.AuthenticationInputDto, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error)
	RefreshTokenFunc  func(ctx context.Context, refreshToken string) (*dtos.AuthData, error)
	LogoutFunc        func(ctx context.Context, refreshToken string) error
	LogoutAllFunc     func(ctx context.Context, refreshToken string) error
	GetSessionsFunc   func(ctx context.Context, refreshToken string) ([]domain.Session, error)
	DeleteSessionFunc func(ctx context.Context, sessionID string, userID string) error
}

func (m *MockAuthService) RequestOtp(ctx context.Context, payload *dtos.RequestOtpDto) error {
	if m.RequestOtpFunc != nil {
		return m.RequestOtpFunc(ctx, payload)
	}
	panic("RequestOtp called unexpectedly")
}

func (m *MockAuthService) Authenticate(ctx context.Context, payload *dtos.AuthenticationInputDto, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error) {
	if m.AuthenticateFunc != nil {
		return m.AuthenticateFunc(ctx, payload, requestInfo)
	}
	panic("Authenticate called unexpectedly")
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*dtos.AuthData, error) {
	if m.RefreshTokenFunc != nil {
		return m.RefreshTokenFunc(ctx, refreshToken)
	}
	panic("RefreshToken called unexpectedly")
}

func (m *MockAuthService) Logout(ctx context.Context, refreshToken string) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, refreshToken)
	}
	panic("Logout called unexpectedly")
}

func (m *MockAuthService) LogoutAll(ctx context.Context, refreshToken string) error {
	if m.LogoutAllFunc != nil {
		return m.LogoutAllFunc(ctx, refreshToken)
	}
	panic("LogoutAll called unexpectedly")
}

func (m *MockAuthService) GetSessions(ctx context.Context, refreshToken string) ([]domain.Session, error) {
	if m.GetSessionsFunc != nil {
		return m.GetSessionsFunc(ctx, refreshToken)
	}
	panic("GetSessions called unexpectedly")
}

func (m *MockAuthService) DeleteSession(ctx context.Context, sessionID string, userID string) error {
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(ctx, sessionID, userID)
	}
	panic("DeleteSession called unexpectedly")
}
