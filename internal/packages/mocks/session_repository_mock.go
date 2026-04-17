package mocks

import (
	"context"

	"github.com/ncondes/fifawcp/internal/domain"
)

type MockSessionRepository struct {
	CreateSessionFunc         func(ctx context.Context, session *domain.Session) error
	GetSessionsFunc           func(ctx context.Context, refreshTokenHash string) ([]domain.Session, error)
	UpdateLastUsedAtFunc      func(ctx context.Context, id string) error
	DeleteSessionFunc         func(ctx context.Context, refreshTokenHash string) error
	DeleteAllSessionsFunc     func(ctx context.Context, refreshTokenHash string) error
	DeleteSessionByIdFunc     func(ctx context.Context, sessionID string, userID string) error
	DeleteExpiredSessionsFunc func(ctx context.Context) (int64, error)
}

func (m *MockSessionRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(ctx, session)
	}
	panic("CreateSession called unexpectedly")
}

func (m *MockSessionRepository) GetSessions(ctx context.Context, refreshTokenHash string) ([]domain.Session, error) {
	if m.GetSessionsFunc != nil {
		return m.GetSessionsFunc(ctx, refreshTokenHash)
	}
	panic("GetSessions called unexpectedly")
}

func (m *MockSessionRepository) UpdateLastUsedAt(ctx context.Context, id string) error {
	if m.UpdateLastUsedAtFunc != nil {
		return m.UpdateLastUsedAtFunc(ctx, id)
	}
	panic("UpdateLastUsedAt called unexpectedly")
}

func (m *MockSessionRepository) DeleteSession(ctx context.Context, refreshTokenHash string) error {
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(ctx, refreshTokenHash)
	}
	panic("DeleteSession called unexpectedly")
}

func (m *MockSessionRepository) DeleteAllSessions(ctx context.Context, refreshTokenHash string) error {
	if m.DeleteAllSessionsFunc != nil {
		return m.DeleteAllSessionsFunc(ctx, refreshTokenHash)
	}
	panic("DeleteAllSessions called unexpectedly")
}

func (m *MockSessionRepository) DeleteSessionById(ctx context.Context, sessionID string, userID string) error {
	if m.DeleteSessionByIdFunc != nil {
		return m.DeleteSessionByIdFunc(ctx, sessionID, userID)
	}
	panic("DeleteSessionById called unexpectedly")
}

func (m *MockSessionRepository) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	if m.DeleteExpiredSessionsFunc != nil {
		return m.DeleteExpiredSessionsFunc(ctx)
	}
	panic("DeleteExpiredSessions called unexpectedly")
}
