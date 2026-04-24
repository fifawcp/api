package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

func newTestOAuthService(
	oauthStorage *mocks.MockOAuthStorage,
	googleOAuth2Client *mocks.MockGoogleOAuth2Client,
	googleIDTokenVerifier *mocks.MockGoogleIDTokenVerifier,
	oauthAccountRepository *mocks.MockOAuthAccountRepository,
	userRepository *mocks.MockUserRepository,
	authService *mocks.MockAuthService,
) OAuthServiceInterface {
	return NewOAuthService(
		oauthStorage,
		googleOAuth2Client,
		googleIDTokenVerifier,
		oauthAccountRepository,
		userRepository,
		authService,
	)
}

// ---------------------------------------------------------------------------
// TestOAuthService_BeginGoogleLogin
// ---------------------------------------------------------------------------
func TestOAuthService_BeginGoogleLogin(t *testing.T) {
	t.Parallel()

	t.Run("returns oauth code URL on success", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			SetOAuthStateFunc: func(ctx context.Context, state string, payload string) error {
				return nil
			},
		}

		googleOAuth2Client := &mocks.MockGoogleOAuth2Client{
			BuildAuthCodeURLFunc: func(state string) string {
				return "test-url"
			},
		}

		s := newTestOAuthService(oauthStorage, googleOAuth2Client, nil, nil, nil, nil)

		redirectURL, err := s.BeginGoogleLogin(context.Background(), "https://example.com")

		assert.NoError(t, err)
		assert.Equal(t, "test-url", redirectURL)
	})

	t.Run("returns error on set oauth state error", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			SetOAuthStateFunc: func(ctx context.Context, state string, payload string) error {
				return errors.New("set oauth state error")
			},
		}

		s := newTestOAuthService(oauthStorage, nil, nil, nil, nil, nil)

		redirectURL, err := s.BeginGoogleLogin(context.Background(), "https://example.com")

		assert.Error(t, err)
		assert.ErrorContains(t, err, "set oauth state error")
		assert.Empty(t, redirectURL)
	})
}

// ---------------------------------------------------------------------------
// TestOAuthService_CompleteGoogleLogin
// ---------------------------------------------------------------------------
func TestOAuthService_CompleteGoogleLogin(t *testing.T) {
	t.Parallel()

	t.Run("returns authentication data and redirect URL on success for a user with an existing OAuth account", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					Sub:           "test-sub",
					Email:         "test-email",
					Name:          "test-name",
					EmailVerified: true,
					GivenName:     "test-given-name",
					FamilyName:    "test-family-name",
					Picture:       "test-picture",
					Provider:      "google",
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return &domain.OAuthAccount{
					Provider:    "google",
					ProviderSub: "test-sub",
					UserID:      "test-user-id",
				}, nil
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIDFunc: func(ctx context.Context, userID string) (*domain.User, error) {
				return &domain.User{
					ID: "test-user-id",
				}, nil
			},
		}

		authService := &mocks.MockAuthService{
			IssueAuthenticationFunc: func(ctx context.Context, user *domain.User, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error) {
				return &dtos.AuthenticationDto{
					User: user,
					Auth: dtos.AuthData{
						AccessToken:  "test-access-token",
						RefreshToken: "test-refresh-token",
					},
				}, nil
			},
		}

		s := newTestOAuthService(
			oauthStorage,
			oauth2Client,
			idTokenVerifier,
			oauthAccountRepository,
			userRepository,
			authService,
		)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.NoError(t, err)
		assert.Equal(t, &dtos.AuthenticationDto{
			User: &domain.User{
				ID: "test-user-id",
			},
			Auth: dtos.AuthData{
				AccessToken:  "test-access-token",
				RefreshToken: "test-refresh-token",
			},
		}, authentication)
		assert.Equal(t, "https://return-to.com", redirectURL)
	})

	t.Run("returns authentication data and redirect URL on success for a user without an existing OAuth account but with an existing user", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					Sub:           "test-sub",
					Email:         "test-email",
					Name:          "test-name",
					EmailVerified: true,
					GivenName:     "test-given-name",
					FamilyName:    "test-family-name",
					Picture:       "test-picture",
					Provider:      "google",
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return nil, domain.ErrOAuthAccountNotFound
			},
			CreateOAuthAccountFunc: func(ctx context.Context, oauthAccount *domain.OAuthAccount) error {
				return nil
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, identifier string) (*domain.User, error) {
				return &domain.User{
					ID: "test-user-id",
				}, nil
			},
		}

		authService := &mocks.MockAuthService{
			IssueAuthenticationFunc: func(ctx context.Context, user *domain.User, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error) {
				return &dtos.AuthenticationDto{
					User: user,
					Auth: dtos.AuthData{
						AccessToken:  "test-access-token",
						RefreshToken: "test-refresh-token",
					},
				}, nil
			},
		}

		s := newTestOAuthService(
			oauthStorage,
			oauth2Client,
			idTokenVerifier,
			oauthAccountRepository,
			userRepository,
			authService,
		)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.NoError(t, err)
		assert.Equal(t, &dtos.AuthenticationDto{
			User: &domain.User{
				ID: "test-user-id",
			},
			Auth: dtos.AuthData{
				AccessToken:  "test-access-token",
				RefreshToken: "test-refresh-token",
			},
		}, authentication)
		assert.Equal(t, "https://return-to.com", redirectURL)
	})

	t.Run("returns authentication data and redirect URL on success for a user without an existing OAuth account and without an existing user", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					Sub:           "test-sub",
					Email:         "test-email",
					Name:          "test-name",
					EmailVerified: true,
					GivenName:     "test-given-name",
					FamilyName:    "test-family-name",
					Picture:       "test-picture",
					Provider:      "google",
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return nil, domain.ErrOAuthAccountNotFound
			},
			CreateUserWithOAuthAccountFunc: func(ctx context.Context, user *domain.User, oauthAccount *domain.OAuthAccount) error {
				return nil
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, identifier string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		authService := &mocks.MockAuthService{
			IssueAuthenticationFunc: func(ctx context.Context, user *domain.User, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error) {
				return &dtos.AuthenticationDto{
					User: user,
					Auth: dtos.AuthData{
						AccessToken:  "test-access-token",
						RefreshToken: "test-refresh-token",
					},
				}, nil
			},
		}

		s := newTestOAuthService(
			oauthStorage,
			oauth2Client,
			idTokenVerifier,
			oauthAccountRepository,
			userRepository,
			authService,
		)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.NoError(t, err)
		assert.Equal(t, "test-given-name", authentication.User.FirstName)
		assert.Equal(t, "test-family-name", authentication.User.LastName)
		assert.Regexp(t, `^google-test-email-\d{1,4}$`, authentication.User.Username)
		assert.Equal(t, "test-email", authentication.User.Email)
		assert.Equal(t, "test-access-token", authentication.Auth.AccessToken)
		assert.Equal(t, "test-refresh-token", authentication.Auth.RefreshToken)
		assert.Equal(t, "https://return-to.com", redirectURL)
	})

	t.Run("returns authentication data with default names when given name and family name are empty", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					Sub:           "test-sub",
					Email:         "newuser@example.com",
					Name:          "",
					EmailVerified: true,
					GivenName:     "",
					FamilyName:    "",
					Picture:       "test-picture",
					Provider:      "google",
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return nil, domain.ErrOAuthAccountNotFound
			},
			CreateUserWithOAuthAccountFunc: func(ctx context.Context, user *domain.User, oauthAccount *domain.OAuthAccount) error {
				return nil
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, identifier string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		authService := &mocks.MockAuthService{
			IssueAuthenticationFunc: func(ctx context.Context, user *domain.User, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error) {
				return &dtos.AuthenticationDto{
					User: user,
					Auth: dtos.AuthData{
						AccessToken:  "test-access-token",
						RefreshToken: "test-refresh-token",
					},
				}, nil
			},
		}

		s := newTestOAuthService(
			oauthStorage,
			oauth2Client,
			idTokenVerifier,
			oauthAccountRepository,
			userRepository,
			authService,
		)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.NoError(t, err)
		assert.Equal(t, "Google", authentication.User.FirstName)
		assert.Equal(t, "User", authentication.User.LastName)
		assert.Equal(t, "newuser@example.com", authentication.User.Email)
		assert.Regexp(t, `^google-newuser-\d{1,4}$`, authentication.User.Username)
		assert.Equal(t, "https://return-to.com", redirectURL)
	})

	t.Run("caps username local part when email local segment exceeds max length", func(t *testing.T) {
		t.Parallel()

		longLocal := strings.Repeat("a", 50)
		email := longLocal + "@example.com"
		cappedLocal := strings.Repeat("a", 38)

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					Sub:           "test-sub-long-email",
					Email:         email,
					Name:          "test-name",
					EmailVerified: true,
					GivenName:     "test-given-name",
					FamilyName:    "test-family-name",
					Picture:       "test-picture",
					Provider:      "google",
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return nil, domain.ErrOAuthAccountNotFound
			},
			CreateUserWithOAuthAccountFunc: func(ctx context.Context, user *domain.User, oauthAccount *domain.OAuthAccount) error {
				return nil
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, identifier string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		authService := &mocks.MockAuthService{
			IssueAuthenticationFunc: func(ctx context.Context, user *domain.User, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error) {
				return &dtos.AuthenticationDto{
					User: user,
					Auth: dtos.AuthData{
						AccessToken:  "test-access-token",
						RefreshToken: "test-refresh-token",
					},
				}, nil
			},
		}

		s := newTestOAuthService(
			oauthStorage,
			oauth2Client,
			idTokenVerifier,
			oauthAccountRepository,
			userRepository,
			authService,
		)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.NoError(t, err)
		assert.Equal(t, strings.ToLower(email), authentication.User.Email)
		assert.Regexp(t, `^google-`+cappedLocal+`-\d{1,4}$`, authentication.User.Username)
		assert.Equal(t, "https://return-to.com", redirectURL)
	})

	t.Run("returns error when get and delete oauth state error", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "", errors.New("get and delete oauth state error")
			},
		}

		s := newTestOAuthService(oauthStorage, nil, nil, nil, nil, nil)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get and delete oauth state error")
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})

	t.Run("returns error when exchange code for verified id token error", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return nil, errors.New("exchange code for verified id token error")
			},
		}

		s := newTestOAuthService(oauthStorage, oauth2Client, nil, nil, nil, nil)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "exchange code for verified id token error")
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})

	t.Run("returns error when get by provider sub error", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return nil, errors.New("get by provider sub error")
			},
		}

		s := newTestOAuthService(oauthStorage, oauth2Client, idTokenVerifier, oauthAccountRepository, nil, nil)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get by provider sub error")
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})

	t.Run("returns error when email not verified", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					EmailVerified: false,
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return nil, domain.ErrOAuthAccountNotFound
			},
		}

		s := newTestOAuthService(oauthStorage, oauth2Client, idTokenVerifier, oauthAccountRepository, nil, nil)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrOAuthAccountNotVerified)
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})

	t.Run("returns error when get user by identifier error", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					EmailVerified: true,
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return nil, domain.ErrOAuthAccountNotFound
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, identifier string) (*domain.User, error) {
				return nil, errors.New("get user by identifier error")
			},
		}

		s := newTestOAuthService(oauthStorage, oauth2Client, idTokenVerifier, oauthAccountRepository, userRepository, nil)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get user by identifier error")
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})

	t.Run("returns error when verify id token error", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return nil, errors.New("verify id token error")
			},
		}

		s := newTestOAuthService(oauthStorage, oauth2Client, idTokenVerifier, nil, nil, nil)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "verify id token error")
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})

	t.Run("returns error when get user by ID error for existing OAuth account", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					Sub:           "test-sub",
					Email:         "test-email",
					EmailVerified: true,
					Provider:      "google",
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return &domain.OAuthAccount{
					Provider:    "google",
					ProviderSub: "test-sub",
					UserID:      "test-user-id",
				}, nil
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIDFunc: func(ctx context.Context, userID string) (*domain.User, error) {
				return nil, errors.New("get user by ID error")
			},
		}

		s := newTestOAuthService(
			oauthStorage,
			oauth2Client,
			idTokenVerifier,
			oauthAccountRepository,
			userRepository,
			nil,
		)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get user by ID error")
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})

	t.Run("returns error when issue authentication error for existing OAuth account", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					Sub:           "test-sub",
					Email:         "test-email",
					EmailVerified: true,
					Provider:      "google",
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return &domain.OAuthAccount{
					Provider:    "google",
					ProviderSub: "test-sub",
					UserID:      "test-user-id",
				}, nil
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIDFunc: func(ctx context.Context, userID string) (*domain.User, error) {
				return &domain.User{ID: "test-user-id"}, nil
			},
		}

		authService := &mocks.MockAuthService{
			IssueAuthenticationFunc: func(ctx context.Context, user *domain.User, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error) {
				return nil, errors.New("issue authentication error")
			},
		}

		s := newTestOAuthService(
			oauthStorage,
			oauth2Client,
			idTokenVerifier,
			oauthAccountRepository,
			userRepository,
			authService,
		)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "issue authentication error")
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})

	t.Run("returns error when create oauth account error", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					Sub:           "test-sub",
					Email:         "test-email",
					EmailVerified: true,
					GivenName:     "test-given-name",
					FamilyName:    "test-family-name",
					Provider:      "google",
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return nil, domain.ErrOAuthAccountNotFound
			},
			CreateOAuthAccountFunc: func(ctx context.Context, oauthAccount *domain.OAuthAccount) error {
				return errors.New("create oauth account error")
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, identifier string) (*domain.User, error) {
				return &domain.User{ID: "test-user-id"}, nil
			},
		}

		s := newTestOAuthService(
			oauthStorage,
			oauth2Client,
			idTokenVerifier,
			oauthAccountRepository,
			userRepository,
			nil,
		)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "create oauth account error")
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})

	t.Run("returns error when issue authentication error after linking oauth account", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					Sub:           "test-sub",
					Email:         "test-email",
					EmailVerified: true,
					GivenName:     "test-given-name",
					FamilyName:    "test-family-name",
					Provider:      "google",
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return nil, domain.ErrOAuthAccountNotFound
			},
			CreateOAuthAccountFunc: func(ctx context.Context, oauthAccount *domain.OAuthAccount) error {
				return nil
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, identifier string) (*domain.User, error) {
				return &domain.User{ID: "test-user-id"}, nil
			},
		}

		authService := &mocks.MockAuthService{
			IssueAuthenticationFunc: func(ctx context.Context, user *domain.User, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error) {
				return nil, errors.New("issue authentication after link error")
			},
		}

		s := newTestOAuthService(
			oauthStorage,
			oauth2Client,
			idTokenVerifier,
			oauthAccountRepository,
			userRepository,
			authService,
		)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "issue authentication after link error")
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})

	t.Run("returns error when create user with oauth account error", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					Sub:           "test-sub",
					Email:         "test-email",
					EmailVerified: true,
					GivenName:     "test-given-name",
					FamilyName:    "test-family-name",
					Provider:      "google",
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return nil, domain.ErrOAuthAccountNotFound
			},
			CreateUserWithOAuthAccountFunc: func(ctx context.Context, user *domain.User, account *domain.OAuthAccount) error {
				return errors.New("create user with oauth account error")
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, identifier string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		s := newTestOAuthService(
			oauthStorage,
			oauth2Client,
			idTokenVerifier,
			oauthAccountRepository,
			userRepository,
			nil,
		)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "create user with oauth account error")
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})

	t.Run("returns error when issue authentication error after registering new user", func(t *testing.T) {
		t.Parallel()

		oauthStorage := &mocks.MockOAuthStorage{
			GetAndDeleteOAuthStateFunc: func(ctx context.Context, state string) (string, error) {
				return "https://return-to.com", nil
			},
		}

		oauth2Client := &mocks.MockGoogleOAuth2Client{
			ExchangeCodeForTokenFunc: func(ctx context.Context, code string) (*domain.OIDCToken, error) {
				return &domain.OIDCToken{
					RawIDToken: "test-raw-id-token",
				}, nil
			},
		}

		idTokenVerifier := &mocks.MockGoogleIDTokenVerifier{
			VerifyFunc: func(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
				return &domain.IDToken{
					Sub:           "test-sub",
					Email:         "test-email",
					EmailVerified: true,
					GivenName:     "test-given-name",
					FamilyName:    "test-family-name",
					Provider:      "google",
				}, nil
			},
		}

		oauthAccountRepository := &mocks.MockOAuthAccountRepository{
			GetByProviderSubFunc: func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
				return nil, domain.ErrOAuthAccountNotFound
			},
			CreateUserWithOAuthAccountFunc: func(ctx context.Context, user *domain.User, account *domain.OAuthAccount) error {
				return nil
			},
		}

		userRepository := &mocks.MockUserRepository{
			GetUserByIdentifierFunc: func(ctx context.Context, identifier string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		authService := &mocks.MockAuthService{
			IssueAuthenticationFunc: func(ctx context.Context, user *domain.User, requestInfo dtos.RequestInfo) (*dtos.AuthenticationDto, error) {
				return nil, errors.New("issue authentication after register error")
			},
		}

		s := newTestOAuthService(
			oauthStorage,
			oauth2Client,
			idTokenVerifier,
			oauthAccountRepository,
			userRepository,
			authService,
		)

		authentication, redirectURL, err := s.CompleteGoogleLogin(
			context.Background(),
			"test-state",
			"test-code",
			dtos.RequestInfo{},
		)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "issue authentication after register error")
		assert.Nil(t, authentication)
		assert.Empty(t, redirectURL)
	})
}
