package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

type MockUserService struct {
	GetUserFunc    func(ctx context.Context, userID string) (*domain.User, error)
	UpdateUserFunc func(ctx context.Context, userID string, payload *dtos.UpdateUserDto) (*domain.User, error)
}

func (m *MockUserService) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, userID)
	}
	panic("GetUser called unexpectedly")
}

func (m *MockUserService) UpdateUser(ctx context.Context, userID string, payload *dtos.UpdateUserDto) (*domain.User, error) {
	if m.UpdateUserFunc != nil {
		return m.UpdateUserFunc(ctx, userID, payload)
	}
	panic("UpdateUser called unexpectedly")
}
