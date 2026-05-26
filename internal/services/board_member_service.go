package services

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

type BoardMemberServiceInterface interface {
	JoinBoard(ctx context.Context, joinCode string, userID string) (int64, error)
	GetBoardMember(ctx context.Context, boardID int64, userID string) (*domain.BoardMember, error)
	GetBoardMembers(ctx context.Context, boardID int64, filters domain.BoardMembersFilters, page, limit int) (*domain.BoardMembersPage, error)
	UpdateBoardMemberRole(ctx context.Context, boardID int64, userID string, role domain.BoardMemberRole, payload dtos.UpdateBoardMemberRoleDto) error
	RemoveBoardMember(ctx context.Context, boardID int64, userID string, role domain.BoardMemberRole) error
	LeaveBoard(ctx context.Context, boardID int64, userID string) error
	TransferOwnership(ctx context.Context, boardID int64, callerUserID, targetUserID string, callerRole domain.BoardMemberRole) error
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
) (int64, error) {
	return s.boardMemberRepository.CreateBoardMember(ctx, joinCode, userID)
}

func (s *BoardMemberService) GetBoardMember(
	ctx context.Context,
	boardID int64,
	userID string,
) (*domain.BoardMember, error) {
	if _, err := s.boardRepository.GetBoardByID(ctx, boardID); err != nil {
		return nil, err
	}

	return s.boardMemberRepository.GetBoardMember(ctx, boardID, userID)
}

func (s *BoardMemberService) GetBoardMembers(
	ctx context.Context,
	boardID int64,
	filters domain.BoardMembersFilters,
	page, limit int,
) (*domain.BoardMembersPage, error) {
	return s.boardRepository.GetBoardMembers(ctx, boardID, filters, page, limit)
}

func (s *BoardMemberService) UpdateBoardMemberRole(
	ctx context.Context,
	boardID int64,
	userID string,
	role domain.BoardMemberRole,
	payload dtos.UpdateBoardMemberRoleDto,
) error {
	if !role.CanManage() {
		return domain.ErrForbidden
	}

	if err := assertNotGlobalBoard(ctx, s.boardRepository, boardID); err != nil {
		return err
	}

	if err := s.assertCanManageTarget(ctx, boardID, userID, role); err != nil {
		return err
	}

	return s.boardMemberRepository.UpdateBoardMemberRole(ctx, boardID, userID, payload.Role)
}

func (s *BoardMemberService) RemoveBoardMember(
	ctx context.Context,
	boardID int64,
	userID string,
	role domain.BoardMemberRole,
) error {
	if !role.CanManage() {
		return domain.ErrForbidden
	}

	if err := assertNotGlobalBoard(ctx, s.boardRepository, boardID); err != nil {
		return err
	}

	if err := s.assertCanManageTarget(ctx, boardID, userID, role); err != nil {
		return err
	}

	return s.boardMemberRepository.RemoveBoardMember(ctx, boardID, userID)
}

// assertCanManageTarget enforces that only an owner may act on an admin. Admins can manage
// members only — they cannot change another admin's role or remove them.
func (s *BoardMemberService) assertCanManageTarget(
	ctx context.Context,
	boardID int64,
	targetUserID string,
	callerRole domain.BoardMemberRole,
) error {
	target, err := s.boardMemberRepository.GetBoardMember(ctx, boardID, targetUserID)
	if err != nil {
		return err
	}

	if target.Role == domain.BoardMemberRoleAdmin && callerRole != domain.BoardMemberRoleOwner {
		return domain.ErrBoardManageAdminForbidden
	}

	return nil
}

func (s *BoardMemberService) LeaveBoard(
	ctx context.Context,
	boardID int64,
	userID string,
) error {
	if err := assertNotGlobalBoard(ctx, s.boardRepository, boardID); err != nil {
		return err
	}

	return s.boardMemberRepository.LeaveBoard(ctx, boardID, userID)
}

func (s *BoardMemberService) TransferOwnership(
	ctx context.Context,
	boardID int64,
	callerUserID, targetUserID string,
	callerRole domain.BoardMemberRole,
) error {
	if callerRole != domain.BoardMemberRoleOwner {
		return domain.ErrForbidden
	}

	if err := assertNotGlobalBoard(ctx, s.boardRepository, boardID); err != nil {
		return err
	}

	if callerUserID == targetUserID {
		return domain.ErrCannotTransferOwnershipToSelf
	}

	return s.boardMemberRepository.TransferOwnership(ctx, boardID, callerUserID, targetUserID)
}
