package mocks

import (
	"context"
	"time"

	"github.com/fifawcp/api/internal/domain"
)

type MockOTPStorage struct {
	SetOTPFunc            func(ctx context.Context, otp *domain.OTP, ttl time.Duration) error
	GetOTPFunc            func(ctx context.Context, identifier string, purpose domain.OTPPurpose) (*domain.OTP, error)
	IncrementAttemptsFunc func(ctx context.Context, identifier string, purpose domain.OTPPurpose) error
	DeleteOTPFunc         func(ctx context.Context, identifier string, purpose domain.OTPPurpose) error
}

func (m *MockOTPStorage) SetOTP(ctx context.Context, otp *domain.OTP, ttl time.Duration) error {
	if m.SetOTPFunc != nil {
		return m.SetOTPFunc(ctx, otp, ttl)
	}
	panic("SetOTP called unexpectedly")
}

func (m *MockOTPStorage) GetOTP(ctx context.Context, identifier string, purpose domain.OTPPurpose) (*domain.OTP, error) {
	if m.GetOTPFunc != nil {
		return m.GetOTPFunc(ctx, identifier, purpose)
	}
	panic("GetOTP called unexpectedly")
}

func (m *MockOTPStorage) IncrementAttempts(ctx context.Context, identifier string, purpose domain.OTPPurpose) error {
	if m.IncrementAttemptsFunc != nil {
		return m.IncrementAttemptsFunc(ctx, identifier, purpose)
	}
	panic("IncrementAttempts called unexpectedly")
}

func (m *MockOTPStorage) DeleteOTP(ctx context.Context, identifier string, purpose domain.OTPPurpose) error {
	if m.DeleteOTPFunc != nil {
		return m.DeleteOTPFunc(ctx, identifier, purpose)
	}
	panic("DeleteOTP called unexpectedly")
}
