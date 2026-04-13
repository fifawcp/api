package domain

import (
	"context"
	"time"
)

type OTPPurpose string

const (
	OTPPurposeRegistration OTPPurpose = "registration"
	OTPPurposeLogin        OTPPurpose = "login"
)

type OTP struct {
	Identifier string     `json:"identifier"`
	Purpose    OTPPurpose `json:"purpose"`
	OTPHash    string     `json:"otp_hash"`
	Attempts   int        `json:"attempts"`
	CreatedAt  time.Time  `json:"created_at"`
}

type OTPStorage interface {
	SetOTP(ctx context.Context, otp *OTP, ttl time.Duration) error
	GetOTP(ctx context.Context, identifier string, purpose OTPPurpose) (*OTP, error)
	IncrementAttempts(ctx context.Context, identifier string, purpose OTPPurpose) error
	DeleteOTP(ctx context.Context, identifier string, purpose OTPPurpose) error
}
