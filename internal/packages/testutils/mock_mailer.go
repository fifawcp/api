package testutils

import (
	"context"

	"github.com/ncondes/fifawcp/internal/domain"
)

type MockMailer struct {
	SendOTPEmailFunc     func(ctx context.Context, to, otp string, purpose domain.OTPPurpose) error
	SendWelcomeEmailFunc func(ctx context.Context, to, firstName string) error
}

func (m *MockMailer) SendOTPEmail(ctx context.Context, to, otp string, purpose domain.OTPPurpose) error {
	if m.SendOTPEmailFunc != nil {
		return m.SendOTPEmailFunc(ctx, to, otp, purpose)
	}

	return nil
}

func (m *MockMailer) SendWelcomeEmail(ctx context.Context, to, firstName string) error {
	if m.SendWelcomeEmailFunc != nil {
		return m.SendWelcomeEmailFunc(ctx, to, firstName)
	}

	return nil
}
