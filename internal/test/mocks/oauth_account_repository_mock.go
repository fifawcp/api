package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockOAuthAccountRepository struct {
	CreateOAuthAccountFunc         func(ctx context.Context, oauthAccount *domain.OAuthAccount) error
	GetByProviderSubFunc           func(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error)
	CreateUserWithOAuthAccountFunc func(ctx context.Context, user *domain.User, account *domain.OAuthAccount) error
}

func (m *MockOAuthAccountRepository) CreateOAuthAccount(ctx context.Context, oauthAccount *domain.OAuthAccount) error {
	if m.CreateOAuthAccountFunc != nil {
		return m.CreateOAuthAccountFunc(ctx, oauthAccount)
	}
	panic("CreateOAuthAccount called unexpectedly")
}

func (m *MockOAuthAccountRepository) GetByProviderSub(ctx context.Context, provider string, providerSub string) (*domain.OAuthAccount, error) {
	if m.GetByProviderSubFunc != nil {
		return m.GetByProviderSubFunc(ctx, provider, providerSub)
	}
	panic("GetByProviderSub called unexpectedly")
}

func (m *MockOAuthAccountRepository) CreateUserWithOAuthAccount(ctx context.Context, user *domain.User, account *domain.OAuthAccount) error {
	if m.CreateUserWithOAuthAccountFunc != nil {
		return m.CreateUserWithOAuthAccountFunc(ctx, user, account)
	}
	panic("CreateUserWithOAuthAccount called unexpectedly")
}
