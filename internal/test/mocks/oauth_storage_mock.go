package mocks

import (
	"context"
)

type MockOAuthStorage struct {
	SetOAuthStateFunc          func(ctx context.Context, state string, payload string) error
	GetAndDeleteOAuthStateFunc func(ctx context.Context, state string) (string, error)
}

func (m *MockOAuthStorage) SetOAuthState(ctx context.Context, state string, payload string) error {
	if m.SetOAuthStateFunc != nil {
		return m.SetOAuthStateFunc(ctx, state, payload)
	}
	panic("SetOAuthState called unexpectedly")
}

func (m *MockOAuthStorage) GetAndDeleteOAuthState(ctx context.Context, state string) (string, error) {
	if m.GetAndDeleteOAuthStateFunc != nil {
		return m.GetAndDeleteOAuthStateFunc(ctx, state)
	}
	panic("GetAndDeleteOAuthState called unexpectedly")
}
