package oauth

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
)

type GoogleIDTokenVerifier struct {
	provider *oidc.Provider
	cfg      config.OAuthConfig
	verifier *oidc.IDTokenVerifier
}

func NewGoogleIDTokenVerifier(
	provider *oidc.Provider,
	cfg config.OAuthConfig,
) domain.IDTokenVerifier {
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	return &GoogleIDTokenVerifier{
		provider: provider,
		cfg:      cfg,
		verifier: verifier,
	}
}

func (v *GoogleIDTokenVerifier) Verify(ctx context.Context, rawIDToken string) (*domain.IDToken, error) {
	IDToken, err := v.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}

	var claims domain.IDToken
	claims.Provider = "google"

	if err := IDToken.Claims(&claims); err != nil {
		return nil, err
	}

	return &claims, nil
}
