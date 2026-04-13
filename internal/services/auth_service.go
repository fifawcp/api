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

	"github.com/ncondes/fifa-world-cup-pickems/internal/domain"
	"github.com/ncondes/fifa-world-cup-pickems/internal/dtos"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/auth"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/config"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/logging"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/middlewares"
)

type AuthServiceInterface interface {
	RequestOtp(ctx context.Context, payload *dtos.RequestOtpDto) error
	Authenticate(ctx context.Context, payload *dtos.AuthenticationInputDto, requestInfo *middlewares.RequestInfo) (*dtos.AuthenticationDto, error)
	RefreshToken(ctx context.Context, payload *dtos.RefreshTokenDto) (*dtos.AuthData, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, refreshToken string) error
	GetSessions(ctx context.Context, refreshToken string) ([]domain.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

type AuthService struct {
	userRepository         domain.UserRepositoryInterface
	sessionRepository      domain.SessionRepositoryInterface
	refreshTokenRepository domain.RefreshTokenRepositoryInterface
	otpRepository          domain.OTPRepositoryInterface
	cfg                    *config.Config
	logger                 logging.Logger
	authenticator          auth.Authenticator
}

func NewAuthService(
	userRepository domain.UserRepositoryInterface,
	sessionRepository domain.SessionRepositoryInterface,
	refreshTokenRepository domain.RefreshTokenRepositoryInterface,
	otpRepository domain.OTPRepositoryInterface,
	logger logging.Logger,
	cfg *config.Config,
	authenticator auth.Authenticator,
) *AuthService {
	return &AuthService{
		userRepository:         userRepository,
		sessionRepository:      sessionRepository,
		refreshTokenRepository: refreshTokenRepository,
		otpRepository:          otpRepository,
		cfg:                    cfg,
		logger:                 logger,
		authenticator:          authenticator,
	}
}

func (s *AuthService) RequestOtp(
	ctx context.Context,
	payload *dtos.RequestOtpDto,
) error {
	// Check cooldown before proceeding
	otp, err := s.otpRepository.GetOTP(ctx, payload.Identifier, *payload.Purpose)
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
	_, err = s.generateAndStoreOTP(ctx, payload.Identifier, *payload.Purpose)
	if err != nil {
		return err
	}

	// TODO: Send OTP via email/SMS

	return nil
}

func (s *AuthService) Authenticate(
	ctx context.Context,
	payload *dtos.AuthenticationInputDto,
	requestInfo *middlewares.RequestInfo,
) (*dtos.AuthenticationDto, error) {
	// Verify OTP
	if err := s.verifyOTP(
		ctx,
		payload.Identifier,
		payload.Purpose,
		payload.OTP,
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
	}

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

	// Delete OTP from Redis
	// In case of error we can ignore it as it will expire anyway
	s.otpRepository.DeleteOTP(ctx, payload.Identifier, payload.Purpose)

	authData := dtos.AuthData{
		AccessToken:  accessTokenResult.Token,
		RefreshToken: refreshTokenResult.Token,
		ExpiresAt:    refreshTokenResult.ExpiresAt,
	}

	authentication := &dtos.AuthenticationDto{
		Auth: authData,
		User: user,
	}

	return authentication, nil
}

func (s *AuthService) RefreshToken(
	ctx context.Context,
	payload *dtos.RefreshTokenDto,
) (*dtos.AuthData, error) {
	// Validate refresh token
	refreshToken, err := s.refreshTokenRepository.GetRefreshTokenByTokenHash(ctx, s.hashToken(payload.RefreshToken))
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

func (s *AuthService) DeleteSession(ctx context.Context, sessionID string) error {
	// TODO: check session belongs to the authenticated user
	// - Maybe through context middleware
	return s.sessionRepository.DeleteSessionById(ctx, sessionID)
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
			s.logger.Warn("Login failed: user not found", "identifier", identifier)
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

	if err := s.otpRepository.SetOTP(ctx, otp, s.cfg.Auth.OTPTTL); err != nil {
		return "", err
	}

	// Log OTP in development for debugging
	if s.cfg.Env == "development" {
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
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return ""
		}
		// Convert the random number to a byte and add it to the OTP
		otp[i] = digits[num.Int64()]
	}

	return string(otp)
}

func (s *AuthService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *AuthService) verifyOTP(
	ctx context.Context,
	identifier string,
	purpose domain.OTPPurpose,
	plainOtp string,
) error {
	// Get OTP from Redis
	otp, err := s.otpRepository.GetOTP(ctx, identifier, purpose)
	if err != nil {
		return err
	}

	// Check attempts
	if otp.Attempts >= s.cfg.Auth.MaxOTPAttempts-1 {
		s.otpRepository.DeleteOTP(ctx, identifier, purpose)
		return domain.ErrOTPTooManyAttempts
	}

	// Verify OTP
	if otp.OTPHash != s.hashToken(plainOtp) {
		s.otpRepository.IncrementAttempts(ctx, identifier, purpose)
		return domain.ErrOTPInvalidOrExpired
	}

	return nil
}
