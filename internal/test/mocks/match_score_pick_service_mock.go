package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockMatchScorePickService struct {
	GetMatchScorePicksByUserFunc  func(ctx context.Context, userID string) ([]*domain.UserMatchScorePick, error)
	SaveMatchScorePickFunc        func(ctx context.Context, userID string, matchID int64, homeScore, awayScore int) error
	GetMemberCompetitionPicksFunc func(ctx context.Context, boardID, competitionID int64, userID string) ([]*domain.Match, []*domain.UserMatchScorePick, error)
	GetBoardMatchPicksFunc        func(ctx context.Context, boardID, matchID int64) (*domain.Match, []*domain.BoardMemberMatchPick, error)
}

func (m *MockMatchScorePickService) GetMatchScorePicksByUser(ctx context.Context, userID string) ([]*domain.UserMatchScorePick, error) {
	if m.GetMatchScorePicksByUserFunc != nil {
		return m.GetMatchScorePicksByUserFunc(ctx, userID)
	}
	panic("GetMatchScorePicksByUser called unexpectedly")
}

func (m *MockMatchScorePickService) SaveMatchScorePick(ctx context.Context, userID string, matchID int64, homeScore, awayScore int) error {
	if m.SaveMatchScorePickFunc != nil {
		return m.SaveMatchScorePickFunc(ctx, userID, matchID, homeScore, awayScore)
	}
	panic("SaveMatchScorePick called unexpectedly")
}

func (m *MockMatchScorePickService) GetMemberCompetitionPicks(ctx context.Context, boardID, competitionID int64, userID string) ([]*domain.Match, []*domain.UserMatchScorePick, error) {
	if m.GetMemberCompetitionPicksFunc != nil {
		return m.GetMemberCompetitionPicksFunc(ctx, boardID, competitionID, userID)
	}
	panic("GetMemberCompetitionPicks called unexpectedly")
}

func (m *MockMatchScorePickService) GetBoardMatchPicks(ctx context.Context, boardID, matchID int64) (*domain.Match, []*domain.BoardMemberMatchPick, error) {
	if m.GetBoardMatchPicksFunc != nil {
		return m.GetBoardMatchPicksFunc(ctx, boardID, matchID)
	}
	panic("GetBoardMatchPicks called unexpectedly")
}
