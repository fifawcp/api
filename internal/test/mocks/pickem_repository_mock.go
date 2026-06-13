package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockPickemRepository struct {
	UpsertGroupPicksFunc         func(ctx context.Context, userID string, picks []*domain.UserGroupPick) error
	GetGroupPicksFunc            func(ctx context.Context, userID string) ([]*domain.UserGroupPick, error)
	GetGroupPicksByGroupFunc     func(ctx context.Context, groupCode string) ([]*domain.UserGroupPick, error)
	UpsertBestThirdsFunc         func(ctx context.Context, userID string, bestThirds []*domain.UserBestThirdPick) error
	GetBestThirdPicksFunc        func(ctx context.Context, userID string) ([]*domain.UserBestThirdPick, error)
	GetBestThirdPicksByTeamsFunc func(ctx context.Context, teamFifaCodes []string) ([]*domain.UserBestThirdPick, error)
	UpsertBracketPicksFunc       func(ctx context.Context, userID string, picks []*domain.UserBracketPick) error
	GetBracketPicksFunc          func(ctx context.Context, userID string) ([]*domain.UserBracketPick, error)
	GetBracketPicksByMatchFunc   func(ctx context.Context, matchID int64) ([]*domain.UserBracketPick, error)
	GetChampionPickFunc          func(ctx context.Context, userID string) (*string, error)
	GetChampionPickCountsFunc    func(ctx context.Context, limit int) ([]*domain.TitleFavorite, error)
	GetUserProgressCountsFunc    func(ctx context.Context, userID string) (domain.PickemProgressCounts, error)
	GetLockedGroupCodesFunc      func(ctx context.Context, userID string) ([]string, error)
	SetGroupLocksFunc            func(ctx context.Context, userID string, lockedCodes []string) error
}

func (m *MockPickemRepository) UpsertGroupPicks(ctx context.Context, userID string, picks []*domain.UserGroupPick) error {
	if m.UpsertGroupPicksFunc != nil {
		return m.UpsertGroupPicksFunc(ctx, userID, picks)
	}
	panic("UpsertGroupPicks called unexpectedly")
}

func (m *MockPickemRepository) GetGroupPicks(ctx context.Context, userID string) ([]*domain.UserGroupPick, error) {
	if m.GetGroupPicksFunc != nil {
		return m.GetGroupPicksFunc(ctx, userID)
	}
	panic("GetGroupPicks called unexpectedly")
}

func (m *MockPickemRepository) GetGroupPicksByGroup(ctx context.Context, groupCode string) ([]*domain.UserGroupPick, error) {
	if m.GetGroupPicksByGroupFunc != nil {
		return m.GetGroupPicksByGroupFunc(ctx, groupCode)
	}
	panic("GetGroupPicksByGroup called unexpectedly")
}

func (m *MockPickemRepository) UpsertBestThirds(ctx context.Context, userID string, bestThirds []*domain.UserBestThirdPick) error {
	if m.UpsertBestThirdsFunc != nil {
		return m.UpsertBestThirdsFunc(ctx, userID, bestThirds)
	}
	panic("UpsertBestThirds called unexpectedly")
}

func (m *MockPickemRepository) GetBestThirdPicks(ctx context.Context, userID string) ([]*domain.UserBestThirdPick, error) {
	if m.GetBestThirdPicksFunc != nil {
		return m.GetBestThirdPicksFunc(ctx, userID)
	}
	panic("GetBestThirdPicks called unexpectedly")
}

func (m *MockPickemRepository) GetBestThirdPicksByTeams(ctx context.Context, teamFifaCodes []string) ([]*domain.UserBestThirdPick, error) {
	if m.GetBestThirdPicksByTeamsFunc != nil {
		return m.GetBestThirdPicksByTeamsFunc(ctx, teamFifaCodes)
	}
	panic("GetBestThirdPicksByTeams called unexpectedly")
}

func (m *MockPickemRepository) UpsertBracketPicks(ctx context.Context, userID string, picks []*domain.UserBracketPick) error {
	if m.UpsertBracketPicksFunc != nil {
		return m.UpsertBracketPicksFunc(ctx, userID, picks)
	}
	panic("UpsertBracketPicks called unexpectedly")
}

func (m *MockPickemRepository) GetBracketPicks(ctx context.Context, userID string) ([]*domain.UserBracketPick, error) {
	if m.GetBracketPicksFunc != nil {
		return m.GetBracketPicksFunc(ctx, userID)
	}
	panic("GetBracketPicks called unexpectedly")
}

func (m *MockPickemRepository) GetBracketPicksByMatch(ctx context.Context, matchID int64) ([]*domain.UserBracketPick, error) {
	if m.GetBracketPicksByMatchFunc != nil {
		return m.GetBracketPicksByMatchFunc(ctx, matchID)
	}
	panic("GetBracketPicksByMatch called unexpectedly")
}

func (m *MockPickemRepository) GetChampionPick(ctx context.Context, userID string) (*string, error) {
	if m.GetChampionPickFunc != nil {
		return m.GetChampionPickFunc(ctx, userID)
	}
	panic("GetChampionPick called unexpectedly")
}

func (m *MockPickemRepository) GetChampionPickCounts(ctx context.Context, limit int) ([]*domain.TitleFavorite, error) {
	if m.GetChampionPickCountsFunc != nil {
		return m.GetChampionPickCountsFunc(ctx, limit)
	}
	panic("GetChampionPickCounts called unexpectedly")
}

func (m *MockPickemRepository) GetUserProgressCounts(ctx context.Context, userID string) (domain.PickemProgressCounts, error) {
	if m.GetUserProgressCountsFunc != nil {
		return m.GetUserProgressCountsFunc(ctx, userID)
	}
	panic("GetUserProgressCounts called unexpectedly")
}

func (m *MockPickemRepository) GetLockedGroupCodes(ctx context.Context, userID string) ([]string, error) {
	if m.GetLockedGroupCodesFunc != nil {
		return m.GetLockedGroupCodesFunc(ctx, userID)
	}
	panic("GetLockedGroupCodes called unexpectedly")
}

func (m *MockPickemRepository) SetGroupLocks(ctx context.Context, userID string, lockedCodes []string) error {
	if m.SetGroupLocksFunc != nil {
		return m.SetGroupLocksFunc(ctx, userID, lockedCodes)
	}
	panic("SetGroupLocks called unexpectedly")
}
