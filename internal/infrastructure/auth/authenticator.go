package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Authenticator interface {
	GenerateToken(userID string, tokenType TokenType) (*TokenResult, error)
	ValidateToken(tokenString string) (*Claims, error)
}

type TokenResult struct {
	Token     string
	ExpiresAt time.Time
}

type Claims struct {
	jwt.RegisteredClaims
}

type TokenType string

const (
	AccessTokenType  TokenType = "access"
	RefreshTokenType TokenType = "refresh"
)
