package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/auth"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/totp"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

func newTestAuthService(
	ur *mocks.MockUserRepository,
	sr *mocks.MockSessionRepository,
	rtr *mocks.MockRefreshTokenRepository,
	os *mocks.MockOTPStorage,
	logger *mocks.MockLogger,
	authenticator *mocks.MockAuthenticator,
	mailer *mocks.MockMailer,
) AuthServiceInterface {
	cfg := &config.Config{
		Env: "testing",
		JWT: config.JWTConfig{
			Secret:             "test-secret",
			RefreshGraceWindow: 10 * time.Second,
		},
		Auth: config.AuthConfig{
			OTPCooldown:        30 * time.Second,
			MaxOTPAttempts:     3,
			SessionTTL:         24 * time.Hour,
			SessionMaxLifetime: 72 * time.Hour,
		},
	}

	return NewAuthService(ur, sr, rtr, os, logger, cfg, authenticator, mailer)
}

// ---------------------------------------------------------------------------
// TestAuthService_RequestOtp
// ---------------------------------------------------------------------------
func TestAuthService_RequestOtp(t *testing.T) {
	t.Parallel()

	loginPurpose := domain.OTPPurposeLogin
	registrationPurpose := domain.OTPPurposeRegistration

	t.Run("returns nil on success (login)", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return nil, errors.New("otp not found")
			},
			SetOTPFunc: func(ctx context.Context, otp *domain.OTP, ttl time.Duration) error {
				assert.Equal(t, identifier, otp.Identifier)
				assert.Equal(t, loginPurpose, otp.Purpose)
				assert.Equal(t, 0, otp.Attempts)
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				assert.Equal(t, identifier, id)
				return &domain.User{ID: gofakeit.UUID(), Email: identifier}, nil
			},
		}

		authenticator := &mocks.MockAuthenticator{}

		mailer := &mocks.MockMailer{
			SendOTPEmailFunc: func(ctx context.Context, to, otp string, purpose domain.OTPPurpose) error {
				assert.Equal(t, identifier, to)
				assert.NotEmpty(t, otp)
				assert.Equal(t, loginPurpose, purpose)
				return nil
			},
		}

		service := newTestAuthService(ur, nil, nil, os, &mocks.MockLogger{}, authenticator, mailer)

		err := service.RequestOtp(context.Background(), &dtos.RequestOtpDto{
			Identifier: identifier,
			Purpose:    &loginPurpose,
		})

		assert.NoError(t, err)
	})

	t.Run("sends OTP to the resolved account email when logging in by username", func(t *testing.T) {
		t.Parallel()

		username := gofakeit.Username()
		accountEmail := gofakeit.Email()

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return nil, errors.New("otp not found")
			},
			SetOTPFunc: func(ctx context.Context, otp *domain.OTP, ttl time.Duration) error {
				// OTP stays keyed by the identifier the user typed, not the email.
				assert.Equal(t, username, otp.Identifier)
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				assert.Equal(t, username, id)
				return &domain.User{ID: gofakeit.UUID(), Email: accountEmail}, nil
			},
		}

		mailer := &mocks.MockMailer{
			SendOTPEmailFunc: func(ctx context.Context, to, otp string, purpose domain.OTPPurpose) error {
				// Must address the user's real email, never the raw username.
				assert.Equal(t, accountEmail, to)
				return nil
			},
		}

		service := newTestAuthService(ur, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, mailer)

		err := service.RequestOtp(context.Background(), &dtos.RequestOtpDto{
			Identifier: username,
			Purpose:    &loginPurpose,
		})

		assert.NoError(t, err)
	})

	t.Run("returns nil on success (registration)", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return nil, errors.New("otp not found")
			},
			SetOTPFunc: func(ctx context.Context, otp *domain.OTP, ttl time.Duration) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		authenticator := &mocks.MockAuthenticator{}
		mailer := &mocks.MockMailer{SendOTPEmailFunc: func(ctx context.Context, to, otp string, purpose domain.OTPPurpose) error { return nil }}

		service := newTestAuthService(ur, nil, nil, os, &mocks.MockLogger{}, authenticator, mailer)

		err := service.RequestOtp(context.Background(), &dtos.RequestOtpDto{
			Identifier: identifier,
			Purpose:    &registrationPurpose,
		})

		assert.NoError(t, err)
	})

	t.Run("returns error when user already exists (registration)", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return nil, errors.New("otp not found")
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return &domain.User{ID: gofakeit.UUID()}, nil
			},
		}

		service := newTestAuthService(ur, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, &mocks.MockMailer{})

		err := service.RequestOtp(context.Background(), &dtos.RequestOtpDto{
			Identifier: identifier,
			Purpose:    &registrationPurpose,
		})

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
	})

	t.Run("propagates repository error when getting user (registration)", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return nil, errors.New("otp not found")
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return nil, errors.New("db connection failed")
			},
		}

		service := newTestAuthService(ur, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, &mocks.MockMailer{})

		err := service.RequestOtp(context.Background(), &dtos.RequestOtpDto{
			Identifier: identifier,
			Purpose:    &registrationPurpose,
		})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "db connection failed")
	})

	t.Run("propagates storage error when setting OTP", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return nil, errors.New("otp not found")
			},
			SetOTPFunc: func(ctx context.Context, otp *domain.OTP, ttl time.Duration) error {
				return errors.New("storage error")
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return &domain.User{ID: gofakeit.UUID()}, nil
			},
		}

		service := newTestAuthService(ur, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, &mocks.MockMailer{})

		err := service.RequestOtp(context.Background(), &dtos.RequestOtpDto{
			Identifier: identifier,
			Purpose:    &loginPurpose,
		})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "storage error")
	})

	t.Run("propagates storage error when sending email", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return nil, errors.New("otp not found")
			},
			SetOTPFunc: func(ctx context.Context, otp *domain.OTP, ttl time.Duration) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return &domain.User{ID: gofakeit.UUID()}, nil
			},
		}

		mailer := &mocks.MockMailer{
			SendOTPEmailFunc: func(ctx context.Context, email, otp string, purpose domain.OTPPurpose) error {
				return errors.New("mail sending failed")
			},
		}

		service := newTestAuthService(ur, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, mailer)

		err := service.RequestOtp(context.Background(), &dtos.RequestOtpDto{
			Identifier: identifier,
			Purpose:    &loginPurpose,
		})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "mail sending failed")
	})

	t.Run("returns invalid credentials when user not found (login)", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return nil, errors.New("otp not found")
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		service := newTestAuthService(ur, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, &mocks.MockMailer{})

		err := service.RequestOtp(context.Background(), &dtos.RequestOtpDto{
			Identifier: identifier,
			Purpose:    &loginPurpose,
		})

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("returns error on cooldown", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		loginPurpose := domain.OTPPurposeLogin

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    loginPurpose,
					CreatedAt:  time.Now().Add(-29 * time.Second),
				}, nil
			},
		}

		service := newTestAuthService(nil, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, &mocks.MockMailer{})

		err := service.RequestOtp(context.Background(), &dtos.RequestOtpDto{
			Identifier: identifier,
			Purpose:    &loginPurpose,
		})

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrOtpCooldown(30*time.Second))
	})
}

// ---------------------------------------------------------------------------
// TestAuthService_Authenticate
// ---------------------------------------------------------------------------
func TestAuthService_Authenticate(t *testing.T) {
	t.Parallel()

	loginPurpose := domain.OTPPurposeLogin

	t.Run("returns authentication data on success (login)", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()
		expectedUser := &domain.User{ID: userID, Email: identifier}
		requestInfo := dtos.RequestInfo{
			DeviceInfo: dtos.DeviceInfo{
				Browser:     "Chrome",
				Platform:    "Web",
				DisplayName: "Desktop",
				OS:          "MacOS",
			},
		}

		// Use same hashing method as the service
		hash := sha256.Sum256([]byte("123456"))
		validHash := hex.EncodeToString(hash[:])

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    loginPurpose,
					OTPHash:    validHash,
					Attempts:   0,
					CreatedAt:  time.Now(),
				}, nil
			},
			DeleteOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return expectedUser, nil
			},
		}

		sr := &mocks.MockSessionRepository{
			CreateSessionFunc: func(ctx context.Context, session *domain.Session) error {
				session.ID = sessionID
				assert.Equal(t, userID, session.UserID)
				return nil
			},
		}

		rtr := &mocks.MockRefreshTokenRepository{
			CreateRefreshTokenFunc: func(ctx context.Context, token *domain.RefreshToken) error {
				assert.Equal(t, userID, token.UserID)
				assert.Equal(t, sessionID, token.SessionID)
				return nil
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				return &auth.TokenResult{
					Token:     "token_" + string(tokenType),
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil
			},
		}

		// Mock the hashToken by setting a valid hash
		service := newTestAuthService(ur, sr, rtr, os, &mocks.MockLogger{}, authenticator, &mocks.MockMailer{})

		// This test is tricky because hashToken is private. For now, we'll skip the actual OTP verification
		// In a real scenario, we'd either make hashToken public or extract it to a separate package
		// t.Skip("hashToken is private - need to refactor for testability")

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    loginPurpose,
			OTP:        "123456",
		}, requestInfo)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedUser, result.User)
	})

	t.Run("returns error on OTP verification failure", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		loginPurpose := domain.OTPPurposeLogin

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return nil, errors.New("otp not found")
			},
		}

		service := newTestAuthService(nil, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, &mocks.MockMailer{})

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    loginPurpose,
			OTP:        "123456",
		}, dtos.RequestInfo{})

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns otp too many attempts error when max attempts reached", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		loginPurpose := domain.OTPPurposeLogin
		otp := "not-the-right-otp"

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    loginPurpose,
					OTPHash:    "some-hash",
					Attempts:   2, // Max is 3, so this should trigger the error
					CreatedAt:  time.Now(),
				}, nil
			},
			DeleteOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		service := newTestAuthService(nil, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, &mocks.MockMailer{})

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    loginPurpose,
			OTP:        otp,
		}, dtos.RequestInfo{})

		assert.Error(t, err)
		assert.Equal(t, domain.ErrOTPTooManyAttempts, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when otp is invalid or expired", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		loginPurpose := domain.OTPPurposeLogin
		otp := "not-the-right-otp"

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    loginPurpose,
					OTPHash:    "some-hash",
					Attempts:   0,
					CreatedAt:  time.Now(),
				}, nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		service := newTestAuthService(nil, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, &mocks.MockMailer{})

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    loginPurpose,
			OTP:        otp,
		}, dtos.RequestInfo{})

		assert.Error(t, err)
		assert.Equal(t, domain.ErrOTPInvalidOrExpired, err)
		assert.Nil(t, result)
	})

	// t.Run("Verify otp returns nil")

	t.Run("returns error when user not found for login", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		loginPurpose := domain.OTPPurposeLogin
		otp := totp.Generate(identifier, "test-secret")

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    loginPurpose,
					OTPHash:    "valid_hash",
					Attempts:   0,
					CreatedAt:  time.Now(),
				}, nil
			},
			DeleteOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		service := newTestAuthService(ur, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, &mocks.MockMailer{})

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    loginPurpose,
			OTP:        otp,
		}, dtos.RequestInfo{})

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
		assert.Nil(t, result)
	})

	t.Run("returns error when user already exists for registration", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		registrationPurpose := domain.OTPPurposeRegistration
		userID := gofakeit.UUID()
		otp := totp.Generate(identifier, "test-secret")

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    registrationPurpose,
					OTPHash:    "valid_hash",
					Attempts:   0,
					CreatedAt:  time.Now(),
				}, nil
			},
			DeleteOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return &domain.User{ID: userID, Email: identifier}, nil
			},
		}

		service := newTestAuthService(ur, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, &mocks.MockMailer{})

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    registrationPurpose,
			OTP:        otp,
			User: &dtos.CreateUserDto{
				Email:    identifier,
				Username: "testuser",
			},
		}, dtos.RequestInfo{})

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
		assert.Nil(t, result)
	})

	t.Run("creates user for registration", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		registrationPurpose := domain.OTPPurposeRegistration
		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()
		otp := totp.Generate(identifier, "test-secret")

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    registrationPurpose,
					OTPHash:    "valid_hash",
					Attempts:   0,
					CreatedAt:  time.Now(),
				}, nil
			},
			DeleteOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
			CreateUserFunc: func(ctx context.Context, user *domain.User) error {
				user.ID = userID
				return nil
			},
		}

		sr := &mocks.MockSessionRepository{
			CreateSessionFunc: func(ctx context.Context, session *domain.Session) error {
				session.ID = sessionID
				return nil
			},
		}

		rtr := &mocks.MockRefreshTokenRepository{
			CreateRefreshTokenFunc: func(ctx context.Context, token *domain.RefreshToken) error {
				return nil
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				return &auth.TokenResult{
					Token:     "token_" + string(tokenType),
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil
			},
		}

		mailer := &mocks.MockMailer{
			SendWelcomeEmailFunc: func(ctx context.Context, email, firstName string) error {
				return nil
			},
		}

		service := newTestAuthService(ur, sr, rtr, os, &mocks.MockLogger{}, authenticator, mailer)

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    registrationPurpose,
			OTP:        otp,
			User: &dtos.CreateUserDto{
				Email:     identifier,
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
			},
		}, dtos.RequestInfo{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.User)
		assert.Equal(t, userID, result.User.ID)
		assert.Equal(t, identifier, result.User.Email)
	})

	t.Run("propagates user repository error during user creation", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		registrationPurpose := domain.OTPPurposeRegistration
		otp := totp.Generate(identifier, "test-secret")

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    registrationPurpose,
					OTPHash:    "valid_hash",
					Attempts:   0,
					CreatedAt:  time.Now(),
				}, nil
			},
			DeleteOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
			CreateUserFunc: func(ctx context.Context, user *domain.User) error {
				return errors.New("database error")
			},
		}

		service := newTestAuthService(ur, nil, nil, os, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, &mocks.MockMailer{})

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    registrationPurpose,
			OTP:        otp,
			User: &dtos.CreateUserDto{
				Email:    identifier,
				Username: "testuser",
			},
		}, dtos.RequestInfo{})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "database error")
		assert.Nil(t, result)
	})

	t.Run("logs error when welcome email fails", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		registrationPurpose := domain.OTPPurposeRegistration
		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()
		otp := totp.Generate(identifier, "test-secret")

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    registrationPurpose,
					OTPHash:    "valid_hash",
					Attempts:   0,
					CreatedAt:  time.Now(),
				}, nil
			},
			DeleteOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
			CreateUserFunc: func(ctx context.Context, user *domain.User) error {
				user.ID = userID
				return nil
			},
		}

		sr := &mocks.MockSessionRepository{
			CreateSessionFunc: func(ctx context.Context, session *domain.Session) error {
				session.ID = sessionID
				return nil
			},
		}

		rtr := &mocks.MockRefreshTokenRepository{
			CreateRefreshTokenFunc: func(ctx context.Context, token *domain.RefreshToken) error {
				return nil
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				return &auth.TokenResult{
					Token:     "token_" + string(tokenType),
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil
			},
		}

		var errorLogged bool
		logger := &mocks.MockLogger{
			ErrorFunc: func(msg string, args ...any) {
				errorLogged = true
			},
		}

		mailer := &mocks.MockMailer{
			SendWelcomeEmailFunc: func(ctx context.Context, email, firstName string) error {
				return errors.New("email service unavailable")
			},
		}

		service := newTestAuthService(ur, sr, rtr, os, logger, authenticator, mailer)

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    registrationPurpose,
			OTP:        otp,
			User: &dtos.CreateUserDto{
				Email:     identifier,
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
			},
		}, dtos.RequestInfo{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, errorLogged, "error should be logged when welcome email fails")
	})

	t.Run("propagates session repository error when creating session", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		loginPurpose := domain.OTPPurposeLogin
		userID := gofakeit.UUID()
		otp := totp.Generate(identifier, "test-secret")

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    loginPurpose,
					OTPHash:    "valid_hash",
					Attempts:   0,
					CreatedAt:  time.Now(),
				}, nil
			},
			DeleteOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return &domain.User{ID: userID, Email: identifier}, nil
			},
		}

		sr := &mocks.MockSessionRepository{
			CreateSessionFunc: func(ctx context.Context, session *domain.Session) error {
				return errors.New("database error")
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				return &auth.TokenResult{
					Token:     "token_" + string(tokenType),
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil
			},
		}

		service := newTestAuthService(ur, sr, nil, os, &mocks.MockLogger{}, authenticator, &mocks.MockMailer{})

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    loginPurpose,
			OTP:        otp,
		}, dtos.RequestInfo{})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "database error")
		assert.Nil(t, result)
	})

	t.Run("propagates authenticator error when generating access token", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		loginPurpose := domain.OTPPurposeLogin
		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()
		otp := totp.Generate(identifier, "test-secret")

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    loginPurpose,
					OTPHash:    "valid_hash",
					Attempts:   0,
					CreatedAt:  time.Now(),
				}, nil
			},
			DeleteOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return &domain.User{ID: userID, Email: identifier}, nil
			},
		}

		sr := &mocks.MockSessionRepository{
			CreateSessionFunc: func(ctx context.Context, session *domain.Session) error {
				session.ID = sessionID
				return nil
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				if tokenType == auth.AccessTokenType {
					return nil, errors.New("token generation failed")
				}
				return &auth.TokenResult{
					Token:     "token_" + string(tokenType),
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil
			},
		}

		service := newTestAuthService(ur, sr, nil, os, &mocks.MockLogger{}, authenticator, &mocks.MockMailer{})

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    loginPurpose,
			OTP:        otp,
		}, dtos.RequestInfo{})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "token generation failed")
		assert.Nil(t, result)
	})

	t.Run("propagates authenticator error when generating refresh token", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		loginPurpose := domain.OTPPurposeLogin
		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()
		otp := totp.Generate(identifier, "test-secret")

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    loginPurpose,
					OTPHash:    "valid_hash",
					Attempts:   0,
					CreatedAt:  time.Now(),
				}, nil
			},
			DeleteOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return &domain.User{ID: userID, Email: identifier}, nil
			},
		}

		sr := &mocks.MockSessionRepository{
			CreateSessionFunc: func(ctx context.Context, session *domain.Session) error {
				session.ID = sessionID
				return nil
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				if tokenType == auth.RefreshTokenType {
					return nil, errors.New("refresh token generation failed")
				}
				return &auth.TokenResult{
					Token:     "token_" + string(tokenType),
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil
			},
		}

		service := newTestAuthService(ur, sr, nil, os, &mocks.MockLogger{}, authenticator, &mocks.MockMailer{})

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    loginPurpose,
			OTP:        otp,
		}, dtos.RequestInfo{})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "refresh token generation failed")
		assert.Nil(t, result)
	})

	t.Run("propagates refresh token repository error", func(t *testing.T) {
		t.Parallel()

		identifier := gofakeit.Email()
		loginPurpose := domain.OTPPurposeLogin
		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()
		otp := totp.Generate(identifier, "test-secret")

		os := &mocks.MockOTPStorage{
			GetOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) (*domain.OTP, error) {
				return &domain.OTP{
					Identifier: identifier,
					Purpose:    loginPurpose,
					OTPHash:    "valid_hash",
					Attempts:   0,
					CreatedAt:  time.Now(),
				}, nil
			},
			DeleteOTPFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, purpose domain.OTPPurpose) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return &domain.User{ID: userID, Email: identifier}, nil
			},
		}

		sr := &mocks.MockSessionRepository{
			CreateSessionFunc: func(ctx context.Context, session *domain.Session) error {
				session.ID = sessionID
				return nil
			},
		}

		rtr := &mocks.MockRefreshTokenRepository{
			CreateRefreshTokenFunc: func(ctx context.Context, token *domain.RefreshToken) error {
				return errors.New("database error")
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				return &auth.TokenResult{
					Token:     "token_" + string(tokenType),
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil
			},
		}

		service := newTestAuthService(ur, sr, rtr, os, &mocks.MockLogger{}, authenticator, &mocks.MockMailer{})

		result, err := service.Authenticate(context.Background(), &dtos.AuthenticationInputDto{
			Identifier: identifier,
			Purpose:    loginPurpose,
			OTP:        otp,
		}, dtos.RequestInfo{})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "database error")
		assert.Nil(t, result)
	})

	t.Run("propagates device info marshal error", func(t *testing.T) {
		t.Parallel()

		// Since DeviceInfo marshals fine with valid structs, it's hard to make json.Marshal fail
		// This test would require a different approach to trigger a marshal error
		t.Skip("json.Marshal rarely fails with valid structs")
	})
}

// ---------------------------------------------------------------------------
// TestAuthService_RefreshToken
// ---------------------------------------------------------------------------
func TestAuthService_RefreshToken(t *testing.T) {
	t.Parallel()

	t.Run("returns new tokens on success", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()
		refreshToken := "refresh_token_value"

		rtr := &mocks.MockRefreshTokenRepository{
			GetRefreshTokenByTokenHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
				return &domain.RefreshToken{
					UserID:    userID,
					SessionID: sessionID,
				}, nil
			},
			RotateRefreshTokenFunc: func(ctx context.Context, oldHash string, newToken *domain.RefreshToken) error {
				assert.Equal(t, userID, newToken.UserID)
				assert.Equal(t, sessionID, newToken.SessionID)
				return nil
			},
		}

		sr := &mocks.MockSessionRepository{
			UpdateLastUsedAtFunc: func(ctx context.Context, id string, slideTo time.Time, maxLifetime time.Duration) (time.Time, error) {
				assert.Equal(t, sessionID, id)
				// expiry slides to now + SessionTTL, bounded by SessionMaxLifetime
				assert.WithinDuration(t, time.Now().Add(24*time.Hour), slideTo, time.Minute)
				assert.Equal(t, 72*time.Hour, maxLifetime)
				return slideTo, nil
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				return &auth.TokenResult{
					Token:     "new_token_" + string(tokenType),
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil
			},
		}

		service := newTestAuthService(nil, sr, rtr, nil, &mocks.MockLogger{}, authenticator, nil)

		result, err := service.RefreshToken(context.Background(), refreshToken)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
	})

	t.Run("succeeds for an already-rotated token within the grace window", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()
		rotatedAt := time.Now().Add(-2 * time.Second) // within the 10s test grace window

		rotated := false
		rtr := &mocks.MockRefreshTokenRepository{
			GetRefreshTokenByTokenHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
				return &domain.RefreshToken{UserID: userID, SessionID: sessionID, RotatedAt: &rotatedAt}, nil
			},
			RotateRefreshTokenFunc: func(ctx context.Context, oldHash string, newToken *domain.RefreshToken) error {
				rotated = true
				return nil
			},
		}

		sr := &mocks.MockSessionRepository{
			UpdateLastUsedAtFunc: func(ctx context.Context, id string, slideTo time.Time, maxLifetime time.Duration) (time.Time, error) {
				return slideTo, nil
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				return &auth.TokenResult{Token: "new_token_" + string(tokenType), ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		}

		service := newTestAuthService(nil, sr, rtr, nil, &mocks.MockLogger{}, authenticator, nil)

		result, err := service.RefreshToken(context.Background(), "token")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.True(t, rotated, "expected the in-grace token to be re-issued")
	})

	t.Run("rejects an already-rotated token past the grace window", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()
		rotatedAt := time.Now().Add(-time.Hour) // well beyond the 10s test grace window

		rtr := &mocks.MockRefreshTokenRepository{
			GetRefreshTokenByTokenHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
				return &domain.RefreshToken{UserID: userID, SessionID: sessionID, RotatedAt: &rotatedAt}, nil
			},
		}

		service := newTestAuthService(nil, nil, rtr, nil, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, nil)

		result, err := service.RefreshToken(context.Background(), "token")

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalidOrExpired)
		assert.Nil(t, result)
	})

	t.Run("returns invalid or expired refresh token error when refresh token not found", func(t *testing.T) {
		t.Parallel()

		rtr := &mocks.MockRefreshTokenRepository{
			GetRefreshTokenByTokenHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
				return nil, domain.ErrRefreshTokenNotFound
			},
		}

		service := newTestAuthService(nil, nil, rtr, nil, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, nil)

		result, err := service.RefreshToken(context.Background(), "invalid_token")

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalidOrExpired)
		assert.Nil(t, result)
	})

	t.Run("propagates refresh token repository error when getting refresh token", func(t *testing.T) {
		t.Parallel()

		rtr := &mocks.MockRefreshTokenRepository{
			GetRefreshTokenByTokenHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
				return nil, errors.New("database error")
			},
		}

		service := newTestAuthService(nil, nil, rtr, nil, &mocks.MockLogger{}, &mocks.MockAuthenticator{}, nil)

		result, err := service.RefreshToken(context.Background(), "token")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, result)
	})

	t.Run("propagates session repository error", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()

		rtr := &mocks.MockRefreshTokenRepository{
			GetRefreshTokenByTokenHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
				return &domain.RefreshToken{UserID: userID, SessionID: sessionID}, nil
			},
			RotateRefreshTokenFunc: func(ctx context.Context, oldHash string, newToken *domain.RefreshToken) error {
				return nil
			},
		}

		sr := &mocks.MockSessionRepository{
			UpdateLastUsedAtFunc: func(ctx context.Context, id string, slideTo time.Time, maxLifetime time.Duration) (time.Time, error) {
				return time.Time{}, errors.New("database error")
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				return &auth.TokenResult{Token: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		}

		service := newTestAuthService(nil, sr, rtr, nil, &mocks.MockLogger{}, authenticator, nil)

		result, err := service.RefreshToken(context.Background(), "token")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, result)
	})

	t.Run("propagates authenticator error (access token)", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()

		rtr := &mocks.MockRefreshTokenRepository{
			GetRefreshTokenByTokenHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
				return &domain.RefreshToken{UserID: userID, SessionID: sessionID}, nil
			},
			RotateRefreshTokenFunc: func(ctx context.Context, oldHash string, newToken *domain.RefreshToken) error {
				return nil
			},
		}

		sr := &mocks.MockSessionRepository{
			UpdateLastUsedAtFunc: func(ctx context.Context, id string, slideTo time.Time, maxLifetime time.Duration) (time.Time, error) {
				return slideTo, nil
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				if tokenType == auth.AccessTokenType {
					return nil, errors.New("generate token failed")
				}

				return &auth.TokenResult{Token: "refresh-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		}

		service := newTestAuthService(nil, sr, rtr, nil, &mocks.MockLogger{}, authenticator, nil)

		result, err := service.RefreshToken(context.Background(), "token")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "generate token failed")
		assert.Nil(t, result)
	})

	t.Run("propagates authenticator error (refresh token)", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()

		rtr := &mocks.MockRefreshTokenRepository{
			GetRefreshTokenByTokenHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
				return &domain.RefreshToken{UserID: userID, SessionID: sessionID}, nil
			},
			RotateRefreshTokenFunc: func(ctx context.Context, oldHash string, newToken *domain.RefreshToken) error {
				return nil
			},
		}

		sr := &mocks.MockSessionRepository{
			UpdateLastUsedAtFunc: func(ctx context.Context, id string, slideTo time.Time, maxLifetime time.Duration) (time.Time, error) {
				return slideTo, nil
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				if tokenType == auth.RefreshTokenType {
					return nil, errors.New("generate token failed")
				}

				return &auth.TokenResult{Token: "access-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		}

		service := newTestAuthService(nil, sr, rtr, nil, &mocks.MockLogger{}, authenticator, nil)

		result, err := service.RefreshToken(context.Background(), "token")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "generate token failed")
		assert.Nil(t, result)
	})

	t.Run("propagates refresh token repository error when rotating refresh token", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		sessionID := gofakeit.UUID()

		rtr := &mocks.MockRefreshTokenRepository{
			GetRefreshTokenByTokenHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
				return &domain.RefreshToken{UserID: userID, SessionID: sessionID}, nil
			},
			RotateRefreshTokenFunc: func(ctx context.Context, oldHash string, newToken *domain.RefreshToken) error {
				return errors.New("rotate refresh token failed")
			},
		}

		sr := &mocks.MockSessionRepository{
			UpdateLastUsedAtFunc: func(ctx context.Context, id string, slideTo time.Time, maxLifetime time.Duration) (time.Time, error) {
				return slideTo, nil
			},
		}

		authenticator := &mocks.MockAuthenticator{
			GenerateTokenFunc: func(uid string, tokenType auth.TokenType) (*auth.TokenResult, error) {
				return &auth.TokenResult{Token: "access-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		}

		service := newTestAuthService(nil, sr, rtr, nil, &mocks.MockLogger{}, authenticator, nil)

		result, err := service.RefreshToken(context.Background(), "token")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rotate refresh token failed")
		assert.Nil(t, result)
	})
}

// ---------------------------------------------------------------------------
// TestAuthService_Logout
// ---------------------------------------------------------------------------
func TestAuthService_Logout(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on success", func(t *testing.T) {
		t.Parallel()

		sr := &mocks.MockSessionRepository{
			DeleteSessionFunc: func(ctx context.Context, hash string) error {
				return nil
			},
		}

		service := newTestAuthService(nil, sr, nil, nil, &mocks.MockLogger{}, nil, nil)

		err := service.Logout(context.Background(), "token")

		assert.NoError(t, err)
	})

	t.Run("propagates session repository error", func(t *testing.T) {
		t.Parallel()

		sr := &mocks.MockSessionRepository{
			DeleteSessionFunc: func(ctx context.Context, hash string) error {
				return errors.New("database error")
			},
		}

		service := newTestAuthService(nil, sr, nil, nil, &mocks.MockLogger{}, nil, nil)

		err := service.Logout(context.Background(), "token")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

// ---------------------------------------------------------------------------
// TestAuthService_LogoutAll
// ---------------------------------------------------------------------------
func TestAuthService_LogoutAll(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on success", func(t *testing.T) {
		t.Parallel()

		sr := &mocks.MockSessionRepository{
			DeleteAllSessionsFunc: func(ctx context.Context, hash string) error {
				return nil
			},
		}

		service := newTestAuthService(nil, sr, nil, nil, &mocks.MockLogger{}, nil, nil)

		err := service.LogoutAll(context.Background(), "token")

		assert.NoError(t, err)
	})

	t.Run("propagates session repository error", func(t *testing.T) {
		t.Parallel()

		sr := &mocks.MockSessionRepository{
			DeleteAllSessionsFunc: func(ctx context.Context, hash string) error {
				return errors.New("database error")
			},
		}

		service := newTestAuthService(nil, sr, nil, nil, &mocks.MockLogger{}, nil, nil)

		err := service.LogoutAll(context.Background(), "token")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

// ---------------------------------------------------------------------------
// TestAuthService_GetSessions
// ---------------------------------------------------------------------------
func TestAuthService_GetSessions(t *testing.T) {
	t.Parallel()

	t.Run("returns sessions on success", func(t *testing.T) {
		t.Parallel()

		expectedSessions := []domain.Session{
			{ID: gofakeit.UUID(), UserID: gofakeit.UUID()},
			{ID: gofakeit.UUID(), UserID: gofakeit.UUID()},
		}

		sr := &mocks.MockSessionRepository{
			GetSessionsFunc: func(ctx context.Context, hash string) ([]domain.Session, error) {
				return expectedSessions, nil
			},
		}

		service := newTestAuthService(nil, sr, nil, nil, &mocks.MockLogger{}, nil, nil)

		result, err := service.GetSessions(context.Background(), "token")

		assert.NoError(t, err)
		assert.Equal(t, expectedSessions, result)
	})

	t.Run("propagates session repository error", func(t *testing.T) {
		t.Parallel()

		sr := &mocks.MockSessionRepository{
			GetSessionsFunc: func(ctx context.Context, hash string) ([]domain.Session, error) {
				return nil, errors.New("database error")
			},
		}

		service := newTestAuthService(nil, sr, nil, nil, &mocks.MockLogger{}, nil, nil)

		result, err := service.GetSessions(context.Background(), "token")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, result)
	})
}

// ---------------------------------------------------------------------------
// TestAuthService_DeleteSession
// ---------------------------------------------------------------------------
func TestAuthService_DeleteSession(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on success", func(t *testing.T) {
		t.Parallel()

		sessionID := gofakeit.UUID()
		userID := gofakeit.UUID()

		sr := &mocks.MockSessionRepository{
			DeleteSessionByIdFunc: func(ctx context.Context, sid string, uid string) error {
				assert.Equal(t, sessionID, sid)
				assert.Equal(t, userID, uid)
				return nil
			},
		}

		service := newTestAuthService(nil, sr, nil, nil, &mocks.MockLogger{}, nil, nil)

		err := service.DeleteSession(context.Background(), sessionID, userID)

		assert.NoError(t, err)
	})

	t.Run("propagates session repository error", func(t *testing.T) {
		t.Parallel()

		sessionID := gofakeit.UUID()
		userID := gofakeit.UUID()

		sr := &mocks.MockSessionRepository{
			DeleteSessionByIdFunc: func(ctx context.Context, sid string, uid string) error {
				return errors.New("database error")
			},
		}

		service := newTestAuthService(nil, sr, nil, nil, &mocks.MockLogger{}, nil, nil)

		err := service.DeleteSession(context.Background(), sessionID, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}
