package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

type OAuthServiceInterface interface {
	BeginGoogleLogin(ctx context.Context, returnTo string) (string, error)
	CompleteGoogleLogin(ctx context.Context, state string, code string, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, string, error)
}

type OAuthService struct {
	googleOAuth2Client     domain.OAuth2Client
	oauthStorage           domain.OAuthStorage
	googleIDTokenVerifier  domain.IDTokenVerifier
	oauthAccountRepository domain.OAuthAccountRepository
	userRepository         domain.UserRepository
	authService            AuthServiceInterface
}

func NewOAuthService(
	oauthStorage domain.OAuthStorage,
	googleOAuth2Client domain.OAuth2Client,
	googleIDTokenVerifier domain.IDTokenVerifier,
	oauthAccountRepository domain.OAuthAccountRepository,
	userRepository domain.UserRepository,
	authService AuthServiceInterface,
) OAuthServiceInterface {
	return &OAuthService{
		googleOAuth2Client:     googleOAuth2Client,
		oauthStorage:           oauthStorage,
		googleIDTokenVerifier:  googleIDTokenVerifier,
		oauthAccountRepository: oauthAccountRepository,
		userRepository:         userRepository,
		authService:            authService,
	}
}

func (s *OAuthService) BeginGoogleLogin(ctx context.Context, returnTo string) (string, error) {
	state, err := generateRandomOAuthState()
	if err != nil {
		return "", err
	}

	if err := s.oauthStorage.SetOAuthState(ctx, state, returnTo); err != nil {
		return "", err
	}

	return s.googleOAuth2Client.BuildAuthCodeURL(state), nil
}

func (s *OAuthService) CompleteGoogleLogin(ctx context.Context, state string, code string, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, string, error) {
	returnTo, err := s.oauthStorage.GetAndDeleteOAuthState(ctx, state)
	if err != nil {
		return nil, "", err
	}

	idToken, err := s.exchangeCodeForVerifiedIDToken(ctx, code)
	if err != nil {
		return nil, "", err
	}

	return s.resolveLoginForIDToken(ctx, idToken, returnTo, requestInfo)
}

func (s *OAuthService) exchangeCodeForVerifiedIDToken(ctx context.Context, code string) (*domain.IDToken, error) {
	authToken, err := s.googleOAuth2Client.ExchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, err
	}

	return s.googleIDTokenVerifier.Verify(ctx, authToken.RawIDToken)
}

func (s *OAuthService) resolveLoginForIDToken(
	ctx context.Context,
	idToken *domain.IDToken,
	returnTo string,
	requestInfo dtos.RequestInfo,
) (*dtos.AuthenticationDto, string, error) {
	oauthAccount, err := s.oauthAccountRepository.GetByProviderSub(ctx, idToken.Provider, idToken.Sub)
	if err != nil && !errors.Is(err, domain.ErrOAuthAccountNotFound) {
		return nil, "", err
	}

	// Path 1: OAuth account already exists → log in directly
	if oauthAccount != nil {
		return s.loginWithExistingOAuthAccount(ctx, oauthAccount, returnTo, requestInfo)
	}

	if !idToken.EmailVerified {
		return nil, "", domain.ErrOAuthAccountNotVerified
	}

	user, err := s.userRepository.GetUserByIdentifier(ctx, idToken.Email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, "", err
	}

	// Path 2: Email matches an existing user → link the account, then log in
	if user != nil {
		return s.linkOAuthAccountToExistingUser(ctx, idToken, user, returnTo, requestInfo)
	}

	// Path 3: Completely new user → register user + account atomically, then log in
	return s.registerNewUserViaOAuth(ctx, idToken, returnTo, requestInfo)
}

func (s *OAuthService) loginWithExistingOAuthAccount(
	ctx context.Context,
	oauthAccount *domain.OAuthAccount,
	returnTo string,
	requestInfo dtos.RequestInfo,
) (*dtos.AuthenticationDto, string, error) {
	user, err := s.userRepository.GetUserByID(ctx, oauthAccount.UserID)
	if err != nil {
		return nil, "", err
	}

	authentication, err := s.authService.IssueAuthentication(ctx, user, requestInfo)
	if err != nil {
		return nil, "", err
	}

	return authentication, returnTo, nil
}

func (s *OAuthService) linkOAuthAccountToExistingUser(
	ctx context.Context,
	idToken *domain.IDToken,
	existingUser *domain.User,
	returnTo string,
	requestInfo dtos.RequestInfo,
) (*dtos.AuthenticationDto, string, error) {
	err := s.oauthAccountRepository.CreateOAuthAccount(ctx, &domain.OAuthAccount{
		Provider:    idToken.Provider,
		ProviderSub: idToken.Sub,
		UserID:      existingUser.ID,
	})
	if err != nil {
		return nil, "", err
	}

	// If issuing authentication fails this will self-heals on the next login
	authentication, err := s.authService.IssueAuthentication(ctx, existingUser, requestInfo)
	if err != nil {
		return nil, "", err
	}

	return authentication, returnTo, nil
}

func (s *OAuthService) registerNewUserViaOAuth(
	ctx context.Context,
	idToken *domain.IDToken,
	returnTo string,
	requestInfo dtos.RequestInfo,
) (*dtos.AuthenticationDto, string, error) {
	user := &domain.User{
		Email:     strings.ToLower(idToken.Email),
		FirstName: idToken.GivenName,
		LastName:  idToken.FamilyName,
		Username:  s.resolveAvailableUsername(ctx, idToken.Email, idToken.Provider),
	}

	account := &domain.OAuthAccount{
		Provider:    idToken.Provider,
		ProviderSub: idToken.Sub,
	}

	if err := s.oauthAccountRepository.CreateUserWithOAuthAccount(ctx, user, account); err != nil {
		return nil, "", err
	}

	authentication, err := s.authService.IssueAuthentication(ctx, user, requestInfo)
	if err != nil {
		return nil, "", err
	}

	return authentication, returnTo, nil
}

// resolveAvailableUsername prefers the clean email local part and only falls
// back to the prefixed format when that handle is already taken.
func (s *OAuthService) resolveAvailableUsername(ctx context.Context, email string, provider string) string {
	base := usernameBaseFromEmail(email)

	if _, err := s.userRepository.GetUserByIdentifier(ctx, base); errors.Is(err, domain.ErrUserNotFound) {
		return base
	}

	return generateUsernameFromEmail(email, provider)
}

// usernameBaseFromEmail returns the lowercased email local part, capped so the
// prefixed fallback still fits in VARCHAR(20):
// <provider-letter>(1) + 3-digit suffix(3) + "-"(1) + base → base max = 15.
func usernameBaseFromEmail(email string) string {
	base := strings.ToLower(strings.SplitN(email, "@", 2)[0])

	const maxBaseLen = 15
	if len(base) > maxBaseLen {
		base = base[:maxBaseLen]
	}

	return base
}

func generateUsernameFromEmail(email string, provider string) string {
	base := usernameBaseFromEmail(email)

	// Random 3-digit suffix to reduce collisions on the same (provider, base) pair
	n, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		return ""
	}

	providerLetter := strings.ToUpper(provider[:1])
	return fmt.Sprintf("%s%03d-%s", providerLetter, n.Int64(), base)
}

func generateRandomOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
