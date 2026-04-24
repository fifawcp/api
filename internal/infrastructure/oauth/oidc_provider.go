package oauth

import (
	"context"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

func NewOIDCProvider(issuer string) (*oidc.Provider, error) {
	// Create a context with a timeout to limit the time we wait for the connection
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	return provider, nil
}
