package services

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

type BoardMemberServiceInterface interface {
	JoinBoard(ctx context.Context, joinCode string, userID string) error
	GetBoardMember(ctx context.Context, boardID string, userID string) (*domain.BoardMember, error)
	GetBoardMembers(ctx context.Context, boardID string) ([]*domain.BoardMember, error)
	UpdateBoardMemberRole(ctx context.Context, boardID string, userID string, role domain.BoardMemberRole, payload dtos.UpdateBoardMemberRoleDto) error
	RemoveBoardMember(ctx context.Context, boardID string, userID string, role domain.BoardMemberRole) error
}

type BoardMemberService struct {
	boardRepository       domain.BoardRepository
	boardMemberRepository domain.BoardMemberRepository
}

func NewBoardMemberService(
	boardRepository domain.BoardRepository,
	boardMemberRepository domain.BoardMemberRepository,
) BoardMemberServiceInterface {
	return &BoardMemberService{
		boardRepository:       boardRepository,
		boardMemberRepository: boardMemberRepository,
	}
}

func (s *BoardMemberService) JoinBoard(
	ctx context.Context,
	joinCode string,
	userID string,
) error {
	return s.boardMemberRepository.CreateBoardMember(ctx, joinCode, userID)
}

func (s *BoardMemberService) GetBoardMember(
	ctx context.Context,
	boardID string,
	userID string,
) (*domain.BoardMember, error) {
	if _, err := s.boardRepository.GetBoardByID(ctx, boardID); err != nil {
		return nil, err
	}

	return s.boardMemberRepository.GetBoardMember(ctx, boardID, userID)
}

func (s *BoardMemberService) GetBoardMembers(
	ctx context.Context,
	boardID string,
) ([]*domain.BoardMember, error) {
	// We don't need to check if the board exists here
	// As the board middleware calls GetBoardMember which will check
	return s.boardMemberRepository.GetBoardMembers(ctx, boardID)
}

func (s *BoardMemberService) UpdateBoardMemberRole(
	ctx context.Context,
	boardID string,
	userID string,
	role domain.BoardMemberRole,
	payload dtos.UpdateBoardMemberRoleDto,
) error {
	if !s.isAdminMember(role) {
		return domain.ErrForbidden
	}

	return s.boardMemberRepository.UpdateBoardMemberRole(ctx, boardID, userID, payload.Role)
}

func (s *BoardMemberService) RemoveBoardMember(
	ctx context.Context,
	boardID string,
	userID string,
	role domain.BoardMemberRole,
) error {
	if !s.isAdminMember(role) {
		return domain.ErrForbidden
	}

	return s.boardMemberRepository.RemoveBoardMember(ctx, boardID, userID)
}

func (s *BoardMemberService) isAdminMember(role domain.BoardMemberRole) bool {
	return role == domain.BoardMemberRoleAdmin
}
