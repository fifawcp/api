package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/auth"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/mailer"
	"github.com/fifawcp/api/internal/infrastructure/totp"
)

type AuthServiceInterface interface {
	RequestOtp(ctx context.Context, payload *dtos.RequestOtpDto) error
	VerifyOTP(ctx context.Context, payload *dtos.VerifyOtpDto) error
	Authenticate(ctx context.Context, payload *dtos.AuthenticationInputDto, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error)
	RefreshToken(ctx context.Context, refreshToken string) (*dtos.AuthData, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, refreshToken string) error
	GetSessions(ctx context.Context, refreshToken string) ([]domain.Session, error)
	DeleteSession(ctx context.Context, sessionID string, userID string) error
	IssueAuthentication(ctx context.Context, user *domain.User, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error)
}

type AuthService struct {
	userRepository         domain.UserRepository
	sessionRepository      domain.SessionRepository
	refreshTokenRepository domain.RefreshTokenRepository
	otpStorage             domain.OTPStorage
	cfg                    *config.Config
	logger                 logging.Logger
	authenticator          auth.Authenticator
	mailer                 mailer.Mailer
}

func NewAuthService(
	userRepository domain.UserRepository,
	sessionRepository domain.SessionRepository,
	refreshTokenRepository domain.RefreshTokenRepository,
	otpStorage domain.OTPStorage,
	logger logging.Logger,
	cfg *config.Config,
	authenticator auth.Authenticator,
	mailer mailer.Mailer,
) *AuthService {
	return &AuthService{
		userRepository:         userRepository,
		sessionRepository:      sessionRepository,
		refreshTokenRepository: refreshTokenRepository,
		otpStorage:             otpStorage,
		cfg:                    cfg,
		logger:                 logger,
		authenticator:          authenticator,
		mailer:                 mailer,
	}
}

func (s *AuthService) RequestOtp(
	ctx context.Context,
	payload *dtos.RequestOtpDto,
) error {
	// Check cooldown before proceeding
	otp, err := s.otpStorage.GetOTP(ctx, payload.Identifier, *payload.Purpose)
	if err == nil {
		if time.Since(otp.CreatedAt) < s.cfg.Auth.OTPCooldown {
			return domain.ErrOtpCooldown(s.cfg.Auth.OTPCooldown)
		}
	}

	// Verify user exists for the given purpose
	if _, err := s.validateUserForPurpose(ctx, payload.Identifier, *payload.Purpose); err != nil {
		return err
	}

	// Generate and store OTP
	plainOtp, err := s.generateAndStoreOTP(ctx, payload.Identifier, *payload.Purpose)
	if err != nil {
		return err
	}

	if err := s.mailer.SendOTPEmail(
		ctx,
		payload.Identifier,
		plainOtp,
		*payload.Purpose,
	); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) VerifyOTP(
	ctx context.Context,
	payload *dtos.VerifyOtpDto,
) error {
	// ! Non-prod env bypass
	if !s.cfg.IsProd() && payload.OTP == totp.Generate(payload.Identifier, s.cfg.JWT.Secret) {
		return nil
	}

	// Get OTP from Redis
	otp, err := s.otpStorage.GetOTP(ctx, payload.Identifier, *payload.Purpose)
	if err != nil {
		return err
	}

	// Check attempts
	if otp.Attempts >= s.cfg.Auth.MaxOTPAttempts-1 {
		// Error can be ignored as it will eventually be deleted by TTL
		s.otpStorage.DeleteOTP(ctx, payload.Identifier, *payload.Purpose)
		return domain.ErrOTPTooManyAttempts
	}

	// Verify OTP
	if otp.OTPHash != s.hashToken(payload.OTP) {
		s.otpStorage.IncrementAttempts(ctx, payload.Identifier, *payload.Purpose)
		return domain.ErrOTPInvalidOrExpired
	}

	return nil
}

func (s *AuthService) Authenticate(
	ctx context.Context,
	payload *dtos.AuthenticationInputDto,
	requestInfo dtos.RequestInfo,
) (*dtos.AuthenticationDto, error) {
	// Verify OTP
	verifyDTO := dtos.VerifyOtpDto{
		Purpose:    &payload.Purpose,
		OTP:        payload.OTP,
		Identifier: payload.Identifier,
	}
	if err := s.VerifyOTP(
		ctx,
		&verifyDTO,
	); err != nil {
		return nil, err
	}

	// Verify user exists for the given purpose
	user, err := s.validateUserForPurpose(ctx, payload.Identifier, payload.Purpose)
	if err != nil {
		return nil, err
	}

	// If registration, create user (user is nil from validation)
	if payload.Purpose == domain.OTPPurposeRegistration {
		user = &domain.User{
			Email:     payload.User.Email,
			Username:  payload.User.Username,
			FirstName: payload.User.FirstName,
			LastName:  payload.User.LastName,
		}

		if err := s.userRepository.CreateUser(ctx, user); err != nil {
			return nil, err
		}

		// Send welcome email (non-blocking failure — don't fail registration if email fails)
		if err := s.mailer.SendWelcomeEmail(ctx, user.Email, user.FirstName); err != nil {
			s.logger.Error(
				"failed to send welcome email",
				"error", err,
				"email", user.Email,
			)
		}
	}

	authentication, err := s.IssueAuthentication(ctx, user, requestInfo)
	if err != nil {
		return nil, err
	}

	// Delete OTP from Redis
	// In case of error we can ignore it as it will expire anyway
	s.otpStorage.DeleteOTP(ctx, payload.Identifier, payload.Purpose)

	return authentication, nil
}

func (s *AuthService) RefreshToken(
	ctx context.Context,
	refreshTokenString string,
) (*dtos.AuthData, error) {
	// Validate refresh token
	refreshToken, err := s.refreshTokenRepository.GetRefreshTokenByTokenHash(ctx, s.hashToken(refreshTokenString))
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return nil, domain.ErrRefreshTokenInvalidOrExpired
		}

		return nil, err
	}

	// Generate access token
	accessTokenResult, err := s.authenticator.GenerateToken(refreshToken.UserID, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshTokenResult, err := s.authenticator.GenerateToken(refreshToken.UserID, auth.RefreshTokenType)
	if err != nil {
		return nil, err
	}

	// Rotate refresh token
	if err := s.refreshTokenRepository.RotateRefreshToken(ctx, refreshToken.TokenHash, &domain.RefreshToken{
		UserID:    refreshToken.UserID,
		SessionID: refreshToken.SessionID,
		TokenHash: s.hashToken(refreshTokenResult.Token),
		ExpiresAt: refreshTokenResult.ExpiresAt,
	}); err != nil {
		return nil, err
	}

	// Update session last_used_at
	if err := s.sessionRepository.UpdateLastUsedAt(ctx, refreshToken.SessionID); err != nil {
		return nil, err
	}

	return &dtos.AuthData{
		AccessToken:  accessTokenResult.Token,
		RefreshToken: refreshTokenResult.Token,
		ExpiresAt:    refreshTokenResult.ExpiresAt,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.sessionRepository.DeleteSession(ctx, s.hashToken(refreshToken))
}

func (s *AuthService) LogoutAll(ctx context.Context, refreshToken string) error {
	return s.sessionRepository.DeleteAllSessions(ctx, s.hashToken(refreshToken))
}

func (s *AuthService) GetSessions(ctx context.Context, refreshToken string) ([]domain.Session, error) {
	return s.sessionRepository.GetSessions(ctx, s.hashToken(refreshToken))
}

func (s *AuthService) DeleteSession(ctx context.Context, sessionID string, userID string) error {
	return s.sessionRepository.DeleteSessionById(ctx, sessionID, userID)
}

func (s *AuthService) IssueAuthentication(ctx context.Context, user *domain.User, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error) {
	// Create session
	deviceInfoJson, err := json.Marshal(requestInfo.DeviceInfo)
	if err != nil {
		return nil, err
	}

	session := &domain.Session{
		UserID:     user.ID,
		DeviceInfo: deviceInfoJson,
		IPAddress:  requestInfo.IPAddress,
		UserAgent:  requestInfo.UserAgent,
		ExpiresAt:  time.Now().Add(s.cfg.Auth.SessionTTL),
	}

	if err := s.sessionRepository.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	// Generate access token
	accessTokenResult, err := s.authenticator.GenerateToken(user.ID, auth.AccessTokenType)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshTokenResult, err := s.authenticator.GenerateToken(user.ID, auth.RefreshTokenType)
	if err != nil {
		return nil, err
	}

	// Store refresh token hash in DB
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		SessionID: session.ID,
		TokenHash: s.hashToken(refreshTokenResult.Token),
		ExpiresAt: refreshTokenResult.ExpiresAt,
	}

	if err := s.refreshTokenRepository.CreateRefreshToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	return &dtos.AuthenticationDto{
		User: user,
		Auth: dtos.AuthData{
			AccessToken:  accessTokenResult.Token,
			RefreshToken: refreshTokenResult.Token,
			ExpiresAt:    refreshTokenResult.ExpiresAt,
		},
	}, nil
}

func (s *AuthService) validateUserForPurpose(
	ctx context.Context,
	identifier string,
	purpose domain.OTPPurpose,
) (*domain.User, error) {
	user, err := s.userRepository.GetUserByIdentifier(ctx, identifier)

	switch purpose {
	case domain.OTPPurposeRegistration:
		if err != nil && err != domain.ErrUserNotFound {
			return nil, err
		}
		if user != nil {
			return nil, domain.ErrUserAlreadyExists
		}

	case domain.OTPPurposeLogin:
		if err != nil {
			return nil, domain.ErrInvalidCredentials
		}
	}

	return user, nil
}

func (s *AuthService) generateAndStoreOTP(
	ctx context.Context,
	identifier string,
	purpose domain.OTPPurpose,
) (string, error) {
	plainOtp := s.generateOTP(6)
	otpHash := s.hashToken(plainOtp)
	otp := &domain.OTP{
		Identifier: identifier,
		Purpose:    purpose,
		OTPHash:    otpHash,
		Attempts:   0,
		CreatedAt:  time.Now(),
	}

	if err := s.otpStorage.SetOTP(ctx, otp, s.cfg.Auth.OTPTTL); err != nil {
		return "", err
	}

	// Log OTP in development for debugging
	if !s.cfg.IsProd() {
		s.logger.Info(
			"[DEBUG] OTP",
			"purpose", purpose,
			"identifier", identifier,
			"otp", plainOtp,
		)
	}

	return plainOtp, nil
}

func (s *AuthService) generateOTP(length int) string {
	const digits = "0123456789"
	otp := make([]byte, length)

	for i := range otp {
		// Generate a random number between 0 and len(digits)
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))

		// Convert the random number to a byte and add it to the OTP
		otp[i] = digits[num.Int64()]
	}

	return string(otp)
}

func (s *AuthService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
