package oauth

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"golang.org/x/oauth2"
)

type GoogleOAuth2Client struct {
	config *oauth2.Config
}

func NewGoogleOAuth2Client(
	provider *oidc.Provider,
	cfg config.OAuthConfig,
) domain.OAuth2Client {
	return &GoogleOAuth2Client{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
			Endpoint:     provider.Endpoint(),
		},
	}
}

func (c *GoogleOAuth2Client) BuildAuthCodeURL(state string) string {
	return c.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (c *GoogleOAuth2Client) ExchangeCodeForToken(ctx context.Context, code string) (*domain.OIDCToken, error) {
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, domain.ErrMissingIDToken
	}

	return &domain.OIDCToken{RawIDToken: rawIDToken}, nil
}
