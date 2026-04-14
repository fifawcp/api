package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupAuthenticator() Authenticator {
	return NewJWTAuthenticator(
		"secret",
		"audience",
		"issuer",
		1*time.Hour,
		24*time.Hour,
	)
}

func TestJWTAuthenticator_GenerateToken(t *testing.T) {
	t.Parallel()

	t.Run("generates an access token result", func(t *testing.T) {
		t.Parallel()

		authenticator := setupAuthenticator()

		result, err := authenticator.GenerateToken("user-id", AccessTokenType)

		assert.NoError(t, err)
		assert.IsType(t, &TokenResult{}, result)
	})

	t.Run("generates a refresh token result", func(t *testing.T) {
		t.Parallel()

		authenticator := setupAuthenticator()

		result, err := authenticator.GenerateToken("user-id", RefreshTokenType)

		assert.NoError(t, err)
		assert.IsType(t, &TokenResult{}, result)
	})
}

func TestJWTAuthenticator_ValidateToken(t *testing.T) {
	t.Parallel()

	t.Run("validates an access token", func(t *testing.T) {
		t.Parallel()

		authenticator := setupAuthenticator()

		result, err := authenticator.GenerateToken("user-id", AccessTokenType)
		assert.NoError(t, err)

		claims, err := authenticator.ValidateToken(result.Token)
		assert.NoError(t, err)
		assert.Equal(t, "user-id", claims.Subject)
		assert.Equal(t, "audience", claims.Audience[0])
		assert.Equal(t, "issuer", claims.Issuer)
	})

	t.Run("returns error for invalid token", func(t *testing.T) {
		t.Parallel()

		authenticator := setupAuthenticator()

		_, err := authenticator.ValidateToken("invalid-token")
		assert.Error(t, err)
	})

}
