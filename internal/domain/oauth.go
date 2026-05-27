package domain

import (
	"context"
	"time"
)

type OIDCToken struct {
	RawIDToken string
}

type IDToken struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	EmailVerified bool   `json:"email_verified"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Provider      string `json:"-"`
}

type OAuthAccount struct {
	ID          string    `json:"id"`
	Provider    string    `json:"provider"`
	ProviderSub string    `json:"provider_sub"`
	UserID      string    `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type IDTokenVerifier interface {
	Verify(ctx context.Context, rawIDToken string) (*IDToken, error)
}

type OAuth2Client interface {
	BuildAuthCodeURL(state string) string
	ExchangeCodeForToken(ctx context.Context, code string) (*OIDCToken, error)
}

type OAuthAccountRepository interface {
	CreateOAuthAccount(ctx context.Context, oauthAccount *OAuthAccount) error
	GetByProviderSub(ctx context.Context, provider string, providerSub string) (*OAuthAccount, error)
	CreateUserWithOAuthAccount(ctx context.Context, user *User, account *OAuthAccount) error
}

type OAuthStorage interface {
	SetOAuthState(ctx context.Context, state string, payload string) error
	GetAndDeleteOAuthState(ctx context.Context, state string) (string, error)
}
