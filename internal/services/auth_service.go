package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/httpctx"
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
	user, err := s.validateUserForPurpose(ctx, payload.Identifier, *payload.Purpose)
	if err != nil {
		return err
	}

	// Generate and store OTP
	plainOtp, err := s.generateAndStoreOTP(ctx, payload.Identifier, *payload.Purpose)
	if err != nil {
		return err
	}

	// If identifier is an email, use it as the recipient email. Otherwise, use the user's email.
	recipientEmail := payload.Identifier
	if user != nil {
		recipientEmail = user.Email
	}

	if err := s.mailer.SendOTPEmail(
		ctx,
		recipientEmail,
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
	if otp.OTPHash != hashToken(payload.OTP) {
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
				fmt.Sprintf("failed to send welcome email to %s", user.Email),
				logging.Error, err.Error(),
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
	tokenHash := hashToken(refreshTokenString)
	refreshToken, err := s.refreshTokenRepository.GetRefreshTokenByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			s.logRefreshOutcome(ctx, "not_found_or_expired", tokenHash, nil)
			return nil, domain.ErrRefreshTokenInvalidOrExpired
		}

		return nil, err
	}

	// A rotated token is still honored inside the grace window so concurrent refreshes
	// don't log the user out; past the window it's invalid.
	if refreshToken.RotatedAt != nil && time.Since(*refreshToken.RotatedAt) > s.cfg.JWT.RefreshGraceWindow {
		s.logRefreshOutcome(ctx, "rotated_past_grace", tokenHash, refreshToken)
		return nil, domain.ErrRefreshTokenInvalidOrExpired
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

	// Slide the session expiry forward (bounded by SessionMaxLifetime) before rotating,
	// so the new refresh token can be capped to the post-slide expiry rather than the
	// stale pre-slide one — otherwise an active session's token would be clamped short.
	sessionExpiresAt, err := s.sessionRepository.UpdateLastUsedAt(
		ctx,
		refreshToken.SessionID,
		time.Now().Add(s.cfg.Auth.SessionTTL),
		s.cfg.Auth.SessionMaxLifetime,
	)
	if err != nil {
		return nil, err
	}

	refreshTokenExpiresAt := refreshTokenResult.ExpiresAt
	if refreshTokenExpiresAt.After(sessionExpiresAt) {
		refreshTokenExpiresAt = sessionExpiresAt
	}

	// Rotate refresh token
	if err := s.refreshTokenRepository.RotateRefreshToken(ctx, refreshToken.TokenHash, &domain.RefreshToken{
		UserID:    refreshToken.UserID,
		SessionID: refreshToken.SessionID,
		TokenHash: hashToken(refreshTokenResult.Token),
		ExpiresAt: refreshTokenExpiresAt,
	}); err != nil {
		return nil, err
	}

	s.logRefreshOutcome(ctx, "success", tokenHash, refreshToken)

	return &dtos.AuthData{
		AccessToken:  accessTokenResult.Token,
		RefreshToken: refreshTokenResult.Token,
		ExpiresAt:    refreshTokenResult.ExpiresAt,
	}, nil
}

// logRefreshOutcome emits one structured line per refresh attempt. rt is nil when no
// live row was found. Only a hash fingerprint is logged, never the raw token.
func (s *AuthService) logRefreshOutcome(ctx context.Context, outcome, tokenHash string, rt *domain.RefreshToken) {
	fields := []any{
		logging.RefreshOutcome, outcome,
		logging.TokenFingerprint, tokenHash[:8],
		logging.GraceWindowMS, s.cfg.JWT.RefreshGraceWindow.Milliseconds(),
	}

	if rt != nil {
		fields = append(fields, logging.UserID, rt.UserID, logging.SessionID, rt.SessionID)
		if !rt.CreatedAt.IsZero() {
			fields = append(fields, logging.TokenAgeMS, time.Since(rt.CreatedAt).Milliseconds())
		}
		if rt.RotatedAt != nil {
			fields = append(fields, logging.RotatedAgeMS, time.Since(*rt.RotatedAt).Milliseconds())
		}
	}

	if diagnostics := httpctx.GetRefreshDiagnostics(ctx); diagnostics != nil {
		if diagnostics.RequestID != "" {
			fields = append(fields, logging.RequestID, diagnostics.RequestID)
		}
		if diagnostics.Source != "" {
			fields = append(fields, logging.RefreshSource, diagnostics.Source)
		}
	}

	// rotated_past_grace is the real anomaly (race lost past the window / reuse);
	// success and the benign not_found_or_expired stay at info to avoid warn noise.
	if outcome == "rotated_past_grace" {
		s.logger.Warn("refresh outcome", fields...)
	} else {
		s.logger.Info("refresh outcome", fields...)
	}
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.sessionRepository.DeleteSession(ctx, hashToken(refreshToken))
}

func (s *AuthService) LogoutAll(ctx context.Context, refreshToken string) error {
	return s.sessionRepository.DeleteAllSessions(ctx, hashToken(refreshToken))
}

func (s *AuthService) GetSessions(ctx context.Context, refreshToken string) ([]domain.Session, error) {
	return s.sessionRepository.GetSessions(ctx, hashToken(refreshToken))
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
		TokenHash: hashToken(refreshTokenResult.Token),
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
	plainOtp := generateOTP(6)
	otpHash := hashToken(plainOtp)
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
			fmt.Sprintf("TOTP issued to identifier: %s for purpose: %s", identifier, string(purpose)),
			"TOTP", plainOtp,
		)
	}

	return plainOtp, nil
}
