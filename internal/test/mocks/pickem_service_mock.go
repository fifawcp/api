package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockPickemService struct {
	GetUserPickemFunc         func(ctx context.Context, userID string) (*domain.UserPickem, error)
	GetChampionPickFunc       func(ctx context.Context, userID string) (*domain.Team, error)
	GetUserPickemProgressFunc func(ctx context.Context, userID string) (*domain.PickemProgress, error)
	SaveGroupPicksFunc        func(ctx context.Context, userID string, picks []*domain.UserGroupPick) error
	SaveBestThirdsFunc        func(ctx context.Context, userID string, teamFifaCodes []string) error
	SaveBracketPicksFunc      func(ctx context.Context, userID string, picks []*domain.UserBracketPick) error
}

func (m *MockPickemService) GetUserPickem(ctx context.Context, userID string) (*domain.UserPickem, error) {
	if m.GetUserPickemFunc != nil {
		return m.GetUserPickemFunc(ctx, userID)
	}
	panic("GetUserPickem called unexpectedly")
}

func (m *MockPickemService) GetChampionPick(ctx context.Context, userID string) (*domain.Team, error) {
	if m.GetChampionPickFunc != nil {
		return m.GetChampionPickFunc(ctx, userID)
	}
	panic("GetChampionPick called unexpectedly")
}

func (m *MockPickemService) GetUserPickemProgress(ctx context.Context, userID string) (*domain.PickemProgress, error) {
	if m.GetUserPickemProgressFunc != nil {
		return m.GetUserPickemProgressFunc(ctx, userID)
	}
	panic("GetUserPickemProgress called unexpectedly")
}

func (m *MockPickemService) SaveGroupPicks(ctx context.Context, userID string, picks []*domain.UserGroupPick) error {
	if m.SaveGroupPicksFunc != nil {
		return m.SaveGroupPicksFunc(ctx, userID, picks)
	}
	panic("SaveGroupPicks called unexpectedly")
}

func (m *MockPickemService) SaveBestThirds(ctx context.Context, userID string, teamFifaCodes []string) error {
	if m.SaveBestThirdsFunc != nil {
		return m.SaveBestThirdsFunc(ctx, userID, teamFifaCodes)
	}
	panic("SaveBestThirds called unexpectedly")
}

func (m *MockPickemService) SaveBracketPicks(ctx context.Context, userID string, picks []*domain.UserBracketPick) error {
	if m.SaveBracketPicksFunc != nil {
		return m.SaveBracketPicksFunc(ctx, userID, picks)
	}
	panic("SaveBracketPicks called unexpectedly")
}
