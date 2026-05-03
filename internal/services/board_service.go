package services

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

type BoardServiceInterface interface {
	CreateBoard(ctx context.Context, payload dtos.CreateBoardDto, userID string) (*domain.Board, error)
	GetUserBoards(ctx context.Context, userID string) ([]*domain.Board, error)
	GetBoardByID(ctx context.Context, boardID string, userID string) (*domain.BoardDetails, error)
	RegenerateJoinCode(ctx context.Context, boardID string) (string, error)
	UpdateBoard(ctx context.Context, boardID string, role domain.BoardMemberRole, payload dtos.UpdateBoardDto) error
	DeleteBoard(ctx context.Context, boardID string, userID string) error
}

type BoardService struct {
	boardRepository domain.BoardRepository
}

func NewBoardService(
	boardRepository domain.BoardRepository,
) BoardServiceInterface {
	return &BoardService{
		boardRepository: boardRepository,
	}
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
			Name:        payload.Name,
			OwnerUserID: userID,
			JoinCode:    joinCode,
		}

		// Single CTE handles board + member + ranking atomically
		if err := s.boardRepository.CreateBoardWithOwner(ctx, board); err != nil {
			switch {
			// If error is unique violation, retry with new code
			case errors.Is(err, domain.ErrBoardAlreadyExists):
				continue
			default:
				return nil, err
			}
		}

		return board, nil
	}

	// If we exhausted all retries, return board already exists error
	return nil, domain.ErrBoardAlreadyExists
}

func (s *BoardService) GetUserBoards(ctx context.Context, userID string) ([]*domain.Board, error) {
	return s.boardRepository.GetUserBoards(ctx, userID)
}

func (s *BoardService) GetBoardByID(ctx context.Context, boardID string, userID string) (*domain.BoardDetails, error) {
	boardDetails, err := s.boardRepository.GetBoardDetails(ctx, boardID)
	if err != nil {
		return nil, err
	}

	boardDetails.Privacy = "private"

	// Surface the requesting user's `joined_at` and `rank` at the top level
	for _, member := range boardDetails.Members {
		if member.UserID == userID {
			boardDetails.JoinedAt = member.JoinedAt
			boardDetails.UserRank = member.Rank
			break
		}
	}

	return boardDetails, nil
}

func (s *BoardService) RegenerateJoinCode(ctx context.Context, boardID string) (string, error) {
	joinCode := s.generateJoinCode()

	if err := s.boardRepository.UpdateJoinCode(ctx, boardID, joinCode); err != nil {
		return "", err
	}

	return joinCode, nil
}

func (s *BoardService) UpdateBoard(
	ctx context.Context,
	boardID string,
	role domain.BoardMemberRole,
	payload dtos.UpdateBoardDto,
) error {
	if !s.isAdminMember(role) {
		return domain.ErrForbidden
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

func (s *BoardService) DeleteBoard(ctx context.Context, boardID string, userID string) error {
	return s.boardRepository.DeleteBoard(ctx, boardID, userID)
}

func (s *BoardService) isAdminMember(role domain.BoardMemberRole) bool {
	return role == domain.BoardMemberRoleAdmin
}
