package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
)

type MockCompetitionRepository struct {
	CreateCompetitionFunc              func(ctx context.Context, competition *domain.Competition) error
	GetBoardCompetitionsFunc           func(ctx context.Context, boardID int64, viewerUserID string) ([]*domain.CompetitionListItem, error)
	GetCompetitionByIDFunc             func(ctx context.Context, boardID, competitionID int64) (*domain.Competition, error)
	DeleteCompetitionFunc              func(ctx context.Context, boardID, competitionID int64) error
	FindMatchCompetitionsByMatchesFunc func(ctx context.Context, matchIDs []int64) ([]int64, error)
	GetGlobalCompetitionsFunc          func(ctx context.Context) (*domain.Competition, *domain.Competition, error)
}

func (m *MockCompetitionRepository) CreateCompetition(ctx context.Context, competition *domain.Competition) error {
	if m.CreateCompetitionFunc != nil {
		return m.CreateCompetitionFunc(ctx, competition)
	}
	panic("CreateCompetition called unexpectedly")
}

func (m *MockCompetitionRepository) GetBoardCompetitions(ctx context.Context, boardID int64, viewerUserID string) ([]*domain.CompetitionListItem, error) {
	if m.GetBoardCompetitionsFunc != nil {
		return m.GetBoardCompetitionsFunc(ctx, boardID, viewerUserID)
	}
	panic("GetBoardCompetitions called unexpectedly")
}

func (m *MockCompetitionRepository) GetCompetitionByID(ctx context.Context, boardID, competitionID int64) (*domain.Competition, error) {
	if m.GetCompetitionByIDFunc != nil {
		return m.GetCompetitionByIDFunc(ctx, boardID, competitionID)
	}
	panic("GetCompetitionByID called unexpectedly")
}

func (m *MockCompetitionRepository) DeleteCompetition(ctx context.Context, boardID, competitionID int64) error {
	if m.DeleteCompetitionFunc != nil {
		return m.DeleteCompetitionFunc(ctx, boardID, competitionID)
	}
	panic("DeleteCompetition called unexpectedly")
}

func (m *MockCompetitionRepository) FindMatchCompetitionsByMatches(ctx context.Context, matchIDs []int64) ([]int64, error) {
	if m.FindMatchCompetitionsByMatchesFunc != nil {
		return m.FindMatchCompetitionsByMatchesFunc(ctx, matchIDs)
	}
	panic("FindMatchCompetitionsByMatches called unexpectedly")
}

func (m *MockCompetitionRepository) GetGlobalCompetitions(ctx context.Context) (*domain.Competition, *domain.Competition, error) {
	if m.GetGlobalCompetitionsFunc != nil {
		return m.GetGlobalCompetitionsFunc(ctx)
	}
	panic("GetGlobalCompetitions called unexpectedly")
}
