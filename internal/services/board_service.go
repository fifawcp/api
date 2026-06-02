package services

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

const boardPreviewMemberSampleSize = 8

type BoardServiceInterface interface {
	CreateBoard(ctx context.Context, payload dtos.CreateBoardDto, userID string) (*domain.Board, error)
	GetUserBoards(ctx context.Context, userID string) ([]*domain.UserBoardListItem, error)
	GetBoardByID(ctx context.Context, boardID int64, userID string) (*domain.BoardDetails, error)
	GetBoardPreview(ctx context.Context, joinCode string) (*domain.BoardPreview, error)
	RegenerateJoinCode(ctx context.Context, boardID int64, role domain.BoardMemberRole) (string, error)
	UpdateBoard(ctx context.Context, boardID int64, role domain.BoardMemberRole, payload dtos.UpdateBoardDto) error
	DeleteBoard(ctx context.Context, boardID int64, role domain.BoardMemberRole) error
}

type BoardService struct {
	boardRepository       domain.BoardRepository
	competitionRepository domain.CompetitionRepository
}

func NewBoardService(
	boardRepository domain.BoardRepository,
	competitionRepository domain.CompetitionRepository,
) BoardServiceInterface {
	return &BoardService{
		boardRepository:       boardRepository,
		competitionRepository: competitionRepository,
	}
}

var allMatchStages = []domain.MatchStageCode{
	domain.MatchStageCodeGroupStage,
	domain.MatchStageCodeRoundOf32,
	domain.MatchStageCodeRoundOf16,
	domain.MatchStageCodeQuarterFinals,
	domain.MatchStageCodeSemiFinals,
	domain.MatchStageCodeThirdPlace,
	domain.MatchStageCodeFinal,
}

func (s *BoardService) CreateBoard(
	ctx context.Context,
	payload dtos.CreateBoardDto,
	userID string,
) (*domain.Board, error) {
	var joinCode string
	maxRetries := 5

	for range maxRetries {
		joinCode = s.generateJoinCode()

		board := &domain.Board{
			Name:     payload.Name,
			JoinCode: &joinCode,
			Privacy:  domain.BoardPrivacyPrivate,
		}

		if err := s.boardRepository.CreateBoard(ctx, board, userID); err != nil {
			switch {
			// If error is unique violation (duplicate join code), retry with new code
			case errors.Is(err, domain.ErrBoardAlreadyExists):
				continue
			default:
				return nil, err
			}
		}

		if err := s.competitionRepository.CreateCompetition(ctx, &domain.Competition{
			BoardID:   board.ID,
			Type:      domain.CompetitionTypePickem,
			Name:      "Pick'em",
			CreatedBy: &userID,
		}); err != nil {
			_ = s.boardRepository.DeleteBoard(ctx, board.ID)
			return nil, err
		}

		if err := s.competitionRepository.CreateCompetition(ctx, &domain.Competition{
			BoardID:   board.ID,
			Type:      domain.CompetitionTypeMatch,
			Name:      "All Matches",
			CreatedBy: &userID,
			Scope:     &domain.CompetitionScope{Stages: allMatchStages},
		}); err != nil {
			_ = s.boardRepository.DeleteBoard(ctx, board.ID)
			return nil, err
		}

		return board, nil
	}

	// If we exhausted all retries, return board already exists error
	return nil, domain.ErrBoardAlreadyExists
}

func (s *BoardService) GetUserBoards(ctx context.Context, userID string) ([]*domain.UserBoardListItem, error) {
	return s.boardRepository.GetUserBoards(ctx, userID)
}

func (s *BoardService) GetBoardByID(
	ctx context.Context,
	boardID int64,
	userID string,
) (*domain.BoardDetails, error) {
	return s.boardRepository.GetBoardDetails(ctx, boardID, userID)
}

func (s *BoardService) GetBoardPreview(ctx context.Context, joinCode string) (*domain.BoardPreview, error) {
	// Join codes are uppercase; normalize so a stray-cased or padded link still resolves.
	code := strings.ToUpper(strings.TrimSpace(joinCode))
	if code == "" {
		return nil, domain.ErrBoardNotFound
	}

	return s.boardRepository.GetBoardPreview(ctx, code, boardPreviewMemberSampleSize)
}

func (s *BoardService) RegenerateJoinCode(ctx context.Context, boardID int64, role domain.BoardMemberRole) (string, error) {
	if !role.CanManage() {
		return "", domain.ErrForbidden
	}

	if err := assertNotGlobalBoard(ctx, s.boardRepository, boardID); err != nil {
		return "", err
	}

	joinCode := s.generateJoinCode()

	if err := s.boardRepository.UpdateJoinCode(ctx, boardID, joinCode); err != nil {
		return "", err
	}

	return joinCode, nil
}

func (s *BoardService) UpdateBoard(
	ctx context.Context,
	boardID int64,
	role domain.BoardMemberRole,
	payload dtos.UpdateBoardDto,
) error {
	if !role.CanManage() {
		return domain.ErrForbidden
	}

	if err := assertNotGlobalBoard(ctx, s.boardRepository, boardID); err != nil {
		return err
	}

	board := &domain.Board{
		Name: payload.Name,
	}

	return s.boardRepository.UpdateBoard(ctx, boardID, board)
}

func (s *BoardService) generateJoinCode() string {
	const (
		charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		length  = 8
	)

	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}

	return string(result)
}

func (s *BoardService) DeleteBoard(ctx context.Context, boardID int64, role domain.BoardMemberRole) error {
	if role != domain.BoardMemberRoleOwner {
		return domain.ErrForbidden
	}

	if err := assertNotGlobalBoard(ctx, s.boardRepository, boardID); err != nil {
		return err
	}

	return s.boardRepository.DeleteBoard(ctx, boardID)
}

func assertNotGlobalBoard(ctx context.Context, boardRepository domain.BoardRepository, boardID int64) error {
	board, err := boardRepository.GetBoardByID(ctx, boardID)
	if err != nil {
		return err
	}

	if board.Privacy == domain.BoardPrivacyGlobal {
		return domain.ErrBoardIsGlobal
	}

	return nil
}
