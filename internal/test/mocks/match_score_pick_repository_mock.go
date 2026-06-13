package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockMatchScorePickRepository struct {
	UpsertMatchScorePickFunc               func(ctx context.Context, pick *domain.UserMatchScorePick) error
	GetMatchScorePicksByUserFunc           func(ctx context.Context, userID string) ([]*domain.UserMatchScorePick, error)
	GetMatchScorePicksByUserAndMatchesFunc func(ctx context.Context, userID string, matchIDs []int64) ([]*domain.UserMatchScorePick, error)
	GetMatchScorePicksByMatchFunc          func(ctx context.Context, matchID int64) ([]*domain.UserMatchScorePick, error)
	CountMatchScorePicksByUserFunc         func(ctx context.Context, userID string) (int, error)
	GetBoardMembersMatchPicksFunc          func(ctx context.Context, boardID, matchID int64) ([]*domain.BoardMemberMatchPick, error)
}

func (m *MockMatchScorePickRepository) UpsertMatchScorePick(ctx context.Context, pick *domain.UserMatchScorePick) error {
	if m.UpsertMatchScorePickFunc != nil {
		return m.UpsertMatchScorePickFunc(ctx, pick)
	}
	panic("UpsertMatchScorePick called unexpectedly")
}

func (m *MockMatchScorePickRepository) GetMatchScorePicksByUser(ctx context.Context, userID string) ([]*domain.UserMatchScorePick, error) {
	if m.GetMatchScorePicksByUserFunc != nil {
		return m.GetMatchScorePicksByUserFunc(ctx, userID)
	}
	panic("GetMatchScorePicksByUser called unexpectedly")
}

func (m *MockMatchScorePickRepository) GetMatchScorePicksByUserAndMatches(ctx context.Context, userID string, matchIDs []int64) ([]*domain.UserMatchScorePick, error) {
	if m.GetMatchScorePicksByUserAndMatchesFunc != nil {
		return m.GetMatchScorePicksByUserAndMatchesFunc(ctx, userID, matchIDs)
	}
	panic("GetMatchScorePicksByUserAndMatches called unexpectedly")
}

func (m *MockMatchScorePickRepository) GetMatchScorePicksByMatch(ctx context.Context, matchID int64) ([]*domain.UserMatchScorePick, error) {
	if m.GetMatchScorePicksByMatchFunc != nil {
		return m.GetMatchScorePicksByMatchFunc(ctx, matchID)
	}
	panic("GetMatchScorePicksByMatch called unexpectedly")
}

func (m *MockMatchScorePickRepository) CountMatchScorePicksByUser(ctx context.Context, userID string) (int, error) {
	if m.CountMatchScorePicksByUserFunc != nil {
		return m.CountMatchScorePicksByUserFunc(ctx, userID)
	}
	panic("CountMatchScorePicksByUser called unexpectedly")
}

func (m *MockMatchScorePickRepository) GetBoardMembersMatchPicks(ctx context.Context, boardID, matchID int64) ([]*domain.BoardMemberMatchPick, error) {
	if m.GetBoardMembersMatchPicksFunc != nil {
		return m.GetBoardMembersMatchPicksFunc(ctx, boardID, matchID)
	}
	panic("GetBoardMembersMatchPicks called unexpectedly")
}
