package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

type MockCompetitionService struct {
	CreateCompetitionFunc    func(ctx context.Context, boardID int64, userID string, role domain.BoardMemberRole, payload dtos.CreateCompetitionDto) (*domain.CompetitionListItem, error)
	GetBoardCompetitionsFunc func(ctx context.Context, boardID int64, viewerUserID string) ([]*domain.CompetitionListItem, error)
	GetLeaderboardFunc       func(ctx context.Context, competitionID int64, page, limit int, q string) (*domain.CompetitionLeaderboardPage, error)
	DeleteCompetitionFunc    func(ctx context.Context, boardID, competitionID int64, role domain.BoardMemberRole) error
}

func (m *MockCompetitionService) CreateCompetition(ctx context.Context, boardID int64, userID string, role domain.BoardMemberRole, payload dtos.CreateCompetitionDto) (*domain.CompetitionListItem, error) {
	if m.CreateCompetitionFunc != nil {
		return m.CreateCompetitionFunc(ctx, boardID, userID, role, payload)
	}
	panic("CreateCompetition called unexpectedly")
}

func (m *MockCompetitionService) GetBoardCompetitions(ctx context.Context, boardID int64, viewerUserID string) ([]*domain.CompetitionListItem, error) {
	if m.GetBoardCompetitionsFunc != nil {
		return m.GetBoardCompetitionsFunc(ctx, boardID, viewerUserID)
	}
	panic("GetBoardCompetitions called unexpectedly")
}

func (m *MockCompetitionService) GetLeaderboard(ctx context.Context, competitionID int64, page, limit int, q string) (*domain.CompetitionLeaderboardPage, error) {
	if m.GetLeaderboardFunc != nil {
		return m.GetLeaderboardFunc(ctx, competitionID, page, limit, q)
	}
	panic("GetLeaderboard called unexpectedly")
}

func (m *MockCompetitionService) DeleteCompetition(ctx context.Context, boardID, competitionID int64, role domain.BoardMemberRole) error {
	if m.DeleteCompetitionFunc != nil {
		return m.DeleteCompetitionFunc(ctx, boardID, competitionID, role)
	}
	panic("DeleteCompetition called unexpectedly")
}
