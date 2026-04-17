package mocks

import (
	"context"

	"github.com/ncondes/fifawcp/internal/domain"
)

type MockUserService struct {
	GetUserFunc func(ctx context.Context, userID string) (*domain.User, error)
}

func (m *MockUserService) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, userID)
	}
	panic("GetUser called unexpectedly")
}
