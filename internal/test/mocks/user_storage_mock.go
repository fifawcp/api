package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockUserStorage struct {
	GetUserFunc func(ctx context.Context, userID string) (*domain.User, error)
	SetUserFunc func(ctx context.Context, user *domain.User) error
}

func (m *MockUserStorage) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, userID)
	}
	panic("GetUser called unexpectedly")
}

func (m *MockUserStorage) SetUser(ctx context.Context, user *domain.User) error {
	if m.SetUserFunc != nil {
		return m.SetUserFunc(ctx, user)
	}
	panic("SetUser called unexpectedly")
}
