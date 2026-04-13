package domain

import (
	"errors"
	"time"
)

// Generic
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrRegistrationFailed = errors.New("registration failed")

// OTP
var ErrOTPInvalidOrExpired = errors.New("otp is invalid or expired, try again")
var ErrOTPTooManyAttempts = errors.New("too many attempts, try again later")

type OtpCooldownError struct {
	Cooldown time.Duration
}

func (e OtpCooldownError) Error() string {
	return "please wait " + e.Cooldown.String() + " before requesting a new code"
}

func ErrOtpCooldown(cooldown time.Duration) error {
	return OtpCooldownError{Cooldown: cooldown}
}

// User
var ErrUserNotFound = errors.New("user not found")
var ErrUserAlreadyExists = errors.New("user already exists")
var ErrUsernameAlreadyExists = errors.New("username is already taken")

// Refresh Token
var ErrRefreshTokenNotFound = errors.New("refresh token not found")
var ErrRefreshTokenInvalidOrExpired = errors.New("refresh token is invalid or expired")

// Session
var ErrSessionNotFound = errors.New("session not found")
var ErrInvalidSessionExpiration = errors.New("invalid session expiration")
var ErrInvalidSessionLastUsed = errors.New("invalid session last used time")
