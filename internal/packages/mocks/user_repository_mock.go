package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockUserRepository struct {
	CreateUserFunc          func(ctx context.Context, user *domain.User) error
	GetUserByIdentifierFunc func(ctx context.Context, identifier string) (*domain.User, error)
	GetUserByIDFunc         func(ctx context.Context, userID string) (*domain.User, error)
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, user)
	}
	panic("CreateUser called unexpectedly")
}

func (m *MockUserRepository) GetUserByIdentifier(ctx context.Context, identifier string) (*domain.User, error) {
	if m.GetUserByIdentifierFunc != nil {
		return m.GetUserByIdentifierFunc(ctx, identifier)
	}
	panic("GetUserByIdentifier called unexpectedly")
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	if m.GetUserByIDFunc != nil {
		return m.GetUserByIDFunc(ctx, userID)
	}
	panic("GetUserByID called unexpectedly")
}
