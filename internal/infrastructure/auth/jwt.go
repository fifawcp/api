package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTAuthenticator struct {
	secret             string
	audience           string
	issuer             string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

func NewJWTAuthenticator(
	secret, audience, issuer string,
	accessTokenExpiry, refreshTokenExpiry time.Duration,
) Authenticator {
	return &JWTAuthenticator{
		secret:             secret,
		audience:           audience,
		issuer:             issuer,
		accessTokenExpiry:  accessTokenExpiry,
		refreshTokenExpiry: refreshTokenExpiry,
	}
}

func (ja *JWTAuthenticator) GenerateToken(
	userID string,
	tokenType TokenType,
) (*TokenResult, error) {
	expiry := ja.accessTokenExpiry
	if tokenType == RefreshTokenType {
		expiry = ja.refreshTokenExpiry
	}

	expiresAt := time.Now().Add(expiry)
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	jti := hex.EncodeToString(b)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   userID,
			Issuer:    ja.issuer,
			Audience:  jwt.ClaimStrings{ja.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(ja.secret))
	if err != nil {
		return nil, err
	}

	return &TokenResult{
		Token:     tokenString,
		ExpiresAt: expiresAt,
	}, nil
}

func (ja *JWTAuthenticator) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		return []byte(ja.secret), nil
	},
		jwt.WithExpirationRequired(),
		jwt.WithAudience(ja.audience),
		jwt.WithIssuer(ja.issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}
