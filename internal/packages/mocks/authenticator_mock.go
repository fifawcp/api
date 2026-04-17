package mocks

import "github.com/ncondes/fifawcp/internal/infrastructure/auth"

type MockAuthenticator struct {
	GenerateTokenFunc func(userID string, tokenType auth.TokenType) (*auth.TokenResult, error)
	ValidateTokenFunc func(tokenString string) (*auth.Claims, error)
}

func (m *MockAuthenticator) GenerateToken(userID string, tokenType auth.TokenType) (*auth.TokenResult, error) {
	if m.GenerateTokenFunc != nil {
		return m.GenerateTokenFunc(userID, tokenType)
	}

	return nil, nil
}

func (m *MockAuthenticator) ValidateToken(tokenString string) (*auth.Claims, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(tokenString)
	}

	return nil, nil
}
