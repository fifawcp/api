package services

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

// Top-N members previewed on each competition card.
const competitionTopPreviewLimit = 3

type CompetitionServiceInterface interface {
	CreateCompetition(ctx context.Context, boardID int64, userID string, role domain.BoardMemberRole, payload dtos.CreateCompetitionDto) (*domain.CompetitionListItem, error)
	GetBoardCompetitions(ctx context.Context, boardID int64, viewerUserID string) ([]*domain.CompetitionListItem, error)
	GetLeaderboard(ctx context.Context, competitionID int64, page, limit int, q, sort, dir string) (*domain.CompetitionLeaderboardPage, error)
	GetBoardSummary(ctx context.Context, boardID int64, page, limit int, q, sort, dir string) (*domain.BoardSummaryPage, error)
	DeleteCompetition(ctx context.Context, boardID, competitionID int64, role domain.BoardMemberRole) error
}

type CompetitionService struct {
	boardRepository            domain.BoardRepository
	competitionRepository      domain.CompetitionRepository
	competitionScoreRepository domain.CompetitionScoreRepository
}

func NewCompetitionService(
	boardRepository domain.BoardRepository,
	competitionRepository domain.CompetitionRepository,
	competitionScoreRepository domain.CompetitionScoreRepository,
) CompetitionServiceInterface {
	return &CompetitionService{
		boardRepository:            boardRepository,
		competitionRepository:      competitionRepository,
		competitionScoreRepository: competitionScoreRepository,
	}
}

func (s *CompetitionService) CreateCompetition(
	ctx context.Context,
	boardID int64,
	userID string,
	role domain.BoardMemberRole,
	payload dtos.CreateCompetitionDto,
) (*domain.CompetitionListItem, error) {
	if !role.CanManage() {
		return nil, domain.ErrCompetitionForbidden
	}

	if err := assertNotGlobalBoard(ctx, s.boardRepository, boardID); err != nil {
		return nil, err
	}

	competition := &domain.Competition{
		BoardID:     boardID,
		Type:        payload.Type,
		Name:        payload.Name,
		CreatedBy:   &userID,
		PickMatchID: payload.MatchID,
	}

	if payload.Scope != nil {
		competition.Scope = &domain.CompetitionScope{
			Stages:        payload.Scope.Stages,
			TeamFifaCodes: payload.Scope.TeamFifaCodes,
		}
	}

	if err := s.competitionRepository.CreateCompetition(ctx, competition); err != nil {
		return nil, err
	}

	return &domain.CompetitionListItem{Competition: *competition}, nil
}

func (s *CompetitionService) GetBoardCompetitions(
	ctx context.Context,
	boardID int64,
	viewerUserID string,
) ([]*domain.CompetitionListItem, error) {
	competitions, err := s.competitionRepository.GetBoardCompetitions(ctx, boardID, viewerUserID)
	if err != nil {
		return nil, err
	}

	previews, err := s.competitionScoreRepository.GetBoardCompetitionPreviews(ctx, boardID, competitionTopPreviewLimit)
	if err != nil {
		return nil, err
	}

	for _, competition := range competitions {
		competition.TopPreview = previews[competition.ID]
		if competition.TopPreview == nil {
			competition.TopPreview = []*domain.CompetitionLeaderboardEntry{}
		}
	}

	return competitions, nil
}

func (s *CompetitionService) GetBoardSummary(
	ctx context.Context,
	boardID int64,
	page, limit int,
	q, sort, dir string,
) (*domain.BoardSummaryPage, error) {
	return s.competitionScoreRepository.GetBoardSummary(ctx, boardID, page, limit, q, sort, dir)
}

func (s *CompetitionService) GetLeaderboard(
	ctx context.Context,
	competitionID int64,
	page, limit int,
	q, sort, dir string,
) (*domain.CompetitionLeaderboardPage, error) {
	return s.competitionScoreRepository.GetLeaderboard(ctx, competitionID, page, limit, q, sort, dir)
}

func (s *CompetitionService) DeleteCompetition(
	ctx context.Context,
	boardID, competitionID int64,
	role domain.BoardMemberRole,
) error {
	if !role.CanManage() {
		return domain.ErrCompetitionForbidden
	}

	if err := assertNotGlobalBoard(ctx, s.boardRepository, boardID); err != nil {
		return err
	}

	if _, err := s.competitionRepository.GetCompetitionByID(ctx, boardID, competitionID); err != nil {
		return err
	}

	return s.competitionRepository.DeleteCompetition(ctx, boardID, competitionID)
}
