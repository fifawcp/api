package handlers

import (
	"net/http"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/services"
)

type BoardHandler struct {
	cfg                *config.Config
	validator          *validator.Validator
	logger             logging.Logger
	boardService       services.BoardServiceInterface
	boardMemberService services.BoardMemberServiceInterface
}

func NewBoardHandler(
	boardService services.BoardServiceInterface,
	boardMemberService services.BoardMemberServiceInterface,
	cfg *config.Config,
	validator *validator.Validator,
	logger logging.Logger,
) *BoardHandler {
	return &BoardHandler{
		boardService:       boardService,
		boardMemberService: boardMemberService,
		cfg:                cfg,
		validator:          validator,
		logger:             logger,
	}
}

// CreateBoard godoc
//
//	@Summary		Create a new board
//	@Description	Creates a new board with the provided details. Requires authentication.
//	@Tags			boards
//	@Accept			json
//	@Produce		json
//	@Param			board	body		dtos.CreateBoardDto	true	"Board creation data"
//	@Success		201		{object}	httpx.Response		"Board created successfully"
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request body or validation error"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards [post]
func (h *BoardHandler) CreateBoard(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	var body dtos.CreateBoardDto

	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	board, err := h.boardService.CreateBoard(r.Context(), body, user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusCreated, board)
}

// GetUserBoards godoc
//
//	@Summary		Get user's boards
//	@Description	Retrieves all boards that the authenticated user is a member of. Requires authentication.
//	@Tags			boards
//	@Produce		json
//	@Success		200	{object}	httpx.Response		"User's boards retrieved successfully"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		500	{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards [get]
func (h *BoardHandler) GetUserBoards(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	boards, err := h.boardService.GetUserBoards(r.Context(), user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, boards)
}

// JoinBoard godoc
//
//	@Summary		Join a board
//	@Description	Joins a board using a join code. Requires authentication.
//	@Tags			boards
//	@Accept			json
//	@Produce		json
//	@Param			joinCode	body		dtos.JoinBoardDto	true	"Join code"
//	@Success		201			{object}	httpx.Response		"Joined board successfully"
//	@Failure		400			{object}	httpx.ErrorResponse	"Invalid request body or validation error"
//	@Failure		401			{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		401			{object}	httpx.ErrorResponse	"Invalid or expired board join code"
//	@Failure		409			{object}	httpx.ErrorResponse	"User is already a member of this board"
//	@Failure		500			{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/join [post]
func (h *BoardHandler) JoinBoard(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())

	var body dtos.JoinBoardDto

	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	boardID, err := h.boardMemberService.JoinBoard(r.Context(), body.JoinCode, user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusCreated, &dtos.JoinBoardResponseDto{
		BoardID: boardID,
	})
}

// GetBoardByID godoc
//
//	@Summary		Get board details
//	@Description	Retrieves board details with member ranking data. Requires authentication and board membership
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path		string				true	"Board ID"
//	@Success		200		{object}	httpx.Response		"Board details with members retrieved successfully"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httpx.ErrorResponse	"Not a member of this board"
//	@Failure		404		{object}	httpx.ErrorResponse	"Board not found"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId} [get]
func (h *BoardHandler) GetBoardByID(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())
	user := httpctx.GetAuthenticatedUser(r.Context())

	board, err := h.boardService.GetBoardByID(r.Context(), boardID, user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, board)
}

// GetBoardMembers godoc
//
//	@Summary		Get board members
//	@Description	Returns the board's members ordered by join date (newest first), paginated by `page` and `limit` query params (defaults: page=1, limit=20, max limit=100).
//	@Description	`search` filters by case-insensitive substring match across first_name, last_name, and username.
//	@Description	Requires authentication and board membership.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path		string				true	"Board ID"
//	@Param			page	query		int					false	"Page number (1-indexed, default 1)"
//	@Param			limit	query		int					false	"Page size (default 20, max 100)"
//	@Param			search	query		string				false	"Search by first_name, last_name, or username (case-insensitive substring)"
//	@Success		200		{object}	httpx.Response		"Board members page retrieved successfully"
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid pagination params"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	httpx.ErrorResponse	"Not a member of this board"
//	@Failure		404		{object}	httpx.ErrorResponse	"Board not found"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/members [get]
func (h *BoardHandler) GetBoardMembers(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())

	page, limit := httpx.ParsePagination(w, r)

	filters, err := parseBoardMembersFilters(r)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	membersPage, err := h.boardMemberService.GetBoardMembers(r.Context(), boardID, filters, page, limit)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithPaginatedData(w, http.StatusOK, membersPage.Members, membersPage.Pagination)
}

func parseBoardMembersFilters(r *http.Request) (domain.BoardMembersFilters, error) {
	return domain.BoardMembersFilters{
		Search: strings.TrimSpace(r.URL.Query().Get("search")),
	}, nil
}

// LeaveBoard godoc
//
//	@Summary		Leave board
//	@Description	Removes the authenticated user from a board. If the user is the owner, leaving is only allowed when they are the only member and deletes the board.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path	string	true	"Board ID"
//	@Success		204		"Left board successfully"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httpx.ErrorResponse	"Owner cannot leave while other members remain"
//	@Failure		404		{object}	httpx.ErrorResponse	"Board or membership not found"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/leave [delete]
func (h *BoardHandler) LeaveBoard(w http.ResponseWriter, r *http.Request) {
	user := httpctx.GetAuthenticatedUser(r.Context())
	boardID := httpctx.GetBoardID(r.Context())

	if err := h.boardMemberService.LeaveBoard(r.Context(), boardID, user.ID); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusNoContent, nil)
}

// RegenerateJoinCode godoc
//
//	@Summary		Regenerate board join code
//	@Description	Regenerates the join code for a specific board. Requires authentication, board membership, and admin role.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path		string				true	"Board ID"
//	@Success		200		{object}	httpx.Response		"Join code regenerated successfully"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httpx.ErrorResponse	"Not a member of this board or insufficient permissions"
//	@Failure		404		{object}	httpx.ErrorResponse	"Board not found"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/regenerate-join-code [post]
func (h *BoardHandler) RegenerateJoinCode(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())
	boardMemberRole := httpctx.GetBoardMemberRole(r.Context())

	joinCode, err := h.boardService.RegenerateJoinCode(r.Context(), boardID, boardMemberRole)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusOK, &dtos.JoinBoardDto{
		JoinCode: joinCode,
	})
}

// UpdateBoard godoc
//
//	@Summary		Update board
//	@Description	Updates a board's name. Requires authentication, board membership, and admin role.
//	@Tags			boards
//	@Accept			json
//	@Produce		json
//	@Param			boardId	path	string				true	"Board ID"
//	@Param			board	body	dtos.UpdateBoardDto	true	"Board update data"
//	@Success		204		"Board updated successfully"
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request body or validation error"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httpx.ErrorResponse	"Not a member of this board or insufficient permissions"
//	@Failure		404		{object}	httpx.ErrorResponse	"Board not found"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId} [patch]
func (h *BoardHandler) UpdateBoard(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())
	boardMemberRole := httpctx.GetBoardMemberRole(r.Context())

	var body dtos.UpdateBoardDto

	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	if err := h.boardService.UpdateBoard(r.Context(), boardID, boardMemberRole, body); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusNoContent, nil)
}

// DeleteBoard godoc
//
//	@Summary		Delete board
//	@Description	Deletes a board. Requires authentication, board membership, and owner role.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path	string	true	"Board ID"
//	@Success		204		"Board deleted successfully"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httpx.ErrorResponse	"Insufficient permissions (not owner)"
//	@Failure		404		{object}	httpx.ErrorResponse	"Board not found"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId} [delete]
func (h *BoardHandler) DeleteBoard(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())
	boardMemberRole := httpctx.GetBoardMemberRole(r.Context())

	if err := h.boardService.DeleteBoard(r.Context(), boardID, boardMemberRole); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusNoContent, nil)
}

// UpdateBoardMemberRole godoc
//
//	@Summary		Update board member role
//	@Description	Updates a board member's role. Requires authentication, board membership, and admin role.
//	@Tags			boards
//	@Accept			json
//	@Produce		json
//	@Param			boardId	path	string							true	"Board ID"
//	@Param			userId	path	string							true	"User ID"
//	@Param			role	body	dtos.UpdateBoardMemberRoleDto	true	"Role update data"
//	@Success		204		"Member role updated successfully"
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request body or validation error"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httpx.ErrorResponse	"Not a member of this board or insufficient permissions"
//	@Failure		403		{object}	httpx.ErrorResponse	"Cannot modify owner's role"
//	@Failure		404		{object}	httpx.ErrorResponse	"Board member not found"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/members/{userId}/role [patch]
func (h *BoardHandler) UpdateBoardMemberRole(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())
	userID := httpctx.GetUserID(r.Context())
	boardMemberRole := httpctx.GetBoardMemberRole(r.Context())

	var body dtos.UpdateBoardMemberRoleDto

	if err := httpx.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	if err := h.boardMemberService.UpdateBoardMemberRole(r.Context(), boardID, userID, boardMemberRole, body); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusNoContent, nil)
}

// RemoveBoardMember godoc
//
//	@Summary		Remove board member
//	@Description	Removes a member from a board. Requires authentication, board membership, and admin role.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path	string	true	"Board ID"
//	@Param			userId	path	string	true	"User ID"
//	@Success		204		"Member removed successfully"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httpx.ErrorResponse	"Not a member of this board or insufficient permissions"
//	@Failure		403		{object}	httpx.ErrorResponse	"Cannot remove board owner"
//	@Failure		404		{object}	httpx.ErrorResponse	"Board member not found"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/members/{userId} [delete]
func (h *BoardHandler) RemoveBoardMember(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())
	userID := httpctx.GetUserID(r.Context())
	boardMemberRole := httpctx.GetBoardMemberRole(r.Context())

	if err := h.boardMemberService.RemoveBoardMember(r.Context(), boardID, userID, boardMemberRole); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusNoContent, nil)
}

// TransferOwnership godoc
//
//	@Summary		Transfer board ownership
//	@Description	Hands ownership of a private board to another member. The caller (current owner) becomes admin; the target member becomes owner. Requires authentication, board membership, and owner role.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path	string	true	"Board ID"
//	@Param			userId	path	string	true	"Target user ID"
//	@Success		204		"Ownership transferred successfully"
//	@Failure		400		{object}	httpx.ErrorResponse	"Cannot transfer ownership to yourself"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httpx.ErrorResponse	"Not a member of this board or insufficient permissions"
//	@Failure		403		{object}	httpx.ErrorResponse	"Operation not allowed on global board"
//	@Failure		404		{object}	httpx.ErrorResponse	"Target board member not found"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/members/{userId}/transfer-ownership [post]
func (h *BoardHandler) TransferOwnership(w http.ResponseWriter, r *http.Request) {
	boardID := httpctx.GetBoardID(r.Context())
	targetUserID := httpctx.GetUserID(r.Context())
	callerUserID := httpctx.GetAuthenticatedUser(r.Context()).ID
	callerRole := httpctx.GetBoardMemberRole(r.Context())

	if err := h.boardMemberService.TransferOwnership(r.Context(), boardID, callerUserID, targetUserID, callerRole); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.RespondWithData(w, http.StatusNoContent, nil)
}
