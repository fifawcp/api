package handlers

import (
	"net/http"

	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/middlewares"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/packages/httputils"
	"github.com/fifawcp/api/internal/services"
)

type BoardHandler struct {
	cfg                 *config.Config
	validator           *validator.Validator
	logger              logging.Logger
	boardService        services.BoardServiceInterface
	boardMemberService  services.BoardMemberServiceInterface
	boardRankingService services.BoardRankingServiceInterface
}

func NewBoardHandler(
	boardService services.BoardServiceInterface,
	boardMemberService services.BoardMemberServiceInterface,
	boardRankingService services.BoardRankingServiceInterface,
	cfg *config.Config,
	validator *validator.Validator,
	logger logging.Logger,
) *BoardHandler {
	return &BoardHandler{
		boardService:        boardService,
		boardMemberService:  boardMemberService,
		boardRankingService: boardRankingService,
		cfg:                 cfg,
		validator:           validator,
		logger:              logger,
	}
}

// CreateBoard godoc
//
//	@Summary		Create a new board
//	@Description	Creates a new board with the provided details. Requires authentication.
//	@Tags			boards
//	@Accept			json
//	@Produce		json
//	@Param			board	body		dtos.CreateBoardDto		true	"Board creation data"
//	@Success		201		{object}	httputils.Response		"Board created successfully"
//	@Failure		400		{object}	httputils.ErrorResponse	"Invalid request body or validation error"
//	@Failure		401		{object}	httputils.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		500		{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards [post]
func (h *BoardHandler) CreateBoard(w http.ResponseWriter, r *http.Request) {
	user := middlewares.GetAuthenticatedUser(r.Context())

	var body dtos.CreateBoardDto

	if err := httputils.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	board, err := h.boardService.CreateBoard(r.Context(), body, user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusCreated, board)
}

// GetUserBoards godoc
//
//	@Summary		Get user's boards
//	@Description	Retrieves all boards that the authenticated user is a member of. Requires authentication.
//	@Tags			boards
//	@Produce		json
//	@Success		200	{object}	httputils.Response		"User's boards retrieved successfully"
//	@Failure		401	{object}	httputils.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		500	{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards [get]
func (h *BoardHandler) GetUserBoards(w http.ResponseWriter, r *http.Request) {
	user := middlewares.GetAuthenticatedUser(r.Context())

	boards, err := h.boardService.GetUserBoards(r.Context(), user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusOK, boards)
}

// JoinBoard godoc
//
//	@Summary		Join a board
//	@Description	Joins a board using a join code. Requires authentication.
//	@Tags			boards
//	@Accept			json
//	@Produce		json
//	@Param			joinCode	body	dtos.JoinBoardDto	true	"Join code"
//	@Success		204			"Joined board successfully"
//	@Failure		400			{object}	httputils.ErrorResponse	"Invalid request body or validation error"
//	@Failure		401			{object}	httputils.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		401			{object}	httputils.ErrorResponse	"Invalid or expired board join code"
//	@Failure		409			{object}	httputils.ErrorResponse	"User is already a member of this board"
//	@Failure		500			{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/join [post]
func (h *BoardHandler) JoinBoard(w http.ResponseWriter, r *http.Request) {
	user := middlewares.GetAuthenticatedUser(r.Context())

	var body dtos.JoinBoardDto

	if err := httputils.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	if err := h.boardMemberService.JoinBoard(r.Context(), body.JoinCode, user.ID); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusNoContent, nil)
}

// GetBoardByID godoc
//
//	@Summary		Get board details
//	@Description	Retrieves details of a specific board. Requires authentication and board membership.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path		string					true	"Board ID"
//	@Success		200		{object}	httputils.Response		"Board details retrieved successfully"
//	@Failure		401		{object}	httputils.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httputils.ErrorResponse	"Not a member of this board"
//	@Failure		404		{object}	httputils.ErrorResponse	"Board not found"
//	@Failure		500		{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId} [get]
func (h *BoardHandler) GetBoardByID(w http.ResponseWriter, r *http.Request) {
	boardID := middlewares.GetBoardID(r.Context())

	board, err := h.boardService.GetBoardByID(r.Context(), boardID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusOK, board)
}

// GetBoardMembers godoc
//
//	@Summary		Get board members
//	@Description	Retrieves all members of a specific board. Requires authentication and board membership.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path		string					true	"Board ID"
//	@Success		200		{object}	httputils.Response		"Board members retrieved successfully"
//	@Failure		401		{object}	httputils.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httputils.ErrorResponse	"Not a member of this board"
//	@Failure		404		{object}	httputils.ErrorResponse	"Board not found"
//	@Failure		500		{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/members [get]
func (h *BoardHandler) GetBoardMembers(w http.ResponseWriter, r *http.Request) {
	boardID := middlewares.GetBoardID(r.Context())

	members, err := h.boardMemberService.GetBoardMembers(r.Context(), boardID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusOK, members)
}

// GetBoardRanking godoc
//
//	@Summary		Get board ranking
//	@Description	Retrieves the internal ranking for a specific board. Requires authentication and board membership.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path		string					true	"Board ID"
//	@Success		200		{object}	httputils.Response		"Board ranking retrieved successfully"
//	@Failure		401		{object}	httputils.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httputils.ErrorResponse	"Not a member of this board"
//	@Failure		404		{object}	httputils.ErrorResponse	"Board not found"
//	@Failure		500		{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/ranking [get]
func (h *BoardHandler) GetBoardRanking(w http.ResponseWriter, r *http.Request) {
	boardID := middlewares.GetBoardID(r.Context())

	ranking, err := h.boardRankingService.GetBoardRanking(r.Context(), boardID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusOK, ranking)
}

// RegenerateJoinCode godoc
//
//	@Summary		Regenerate board join code
//	@Description	Regenerates the join code for a specific board. Requires authentication, board membership, and admin role.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path		string					true	"Board ID"
//	@Success		200		{object}	httputils.Response		"Join code regenerated successfully"
//	@Failure		401		{object}	httputils.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httputils.ErrorResponse	"Not a member of this board or insufficient permissions"
//	@Failure		404		{object}	httputils.ErrorResponse	"Board not found"
//	@Failure		500		{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/regenerate-join-code [post]
func (h *BoardHandler) RegenerateJoinCode(w http.ResponseWriter, r *http.Request) {
	boardID := middlewares.GetBoardID(r.Context())

	joinCode, err := h.boardService.RegenerateJoinCode(r.Context(), boardID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusOK, &dtos.JoinBoardDto{
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
//	@Failure		400		{object}	httputils.ErrorResponse	"Invalid request body or validation error"
//	@Failure		401		{object}	httputils.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httputils.ErrorResponse	"Not a member of this board or insufficient permissions"
//	@Failure		404		{object}	httputils.ErrorResponse	"Board not found"
//	@Failure		500		{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId} [patch]
func (h *BoardHandler) UpdateBoard(w http.ResponseWriter, r *http.Request) {
	boardID := middlewares.GetBoardID(r.Context())
	boardMemberRole := middlewares.GetBoardMemberRole(r.Context())

	var body dtos.UpdateBoardDto

	if err := httputils.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	if err := h.boardService.UpdateBoard(r.Context(), boardID, boardMemberRole, body); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusNoContent, nil)
}

// DeleteBoard godoc
//
//	@Summary		Delete board
//	@Description	Deletes a board. Requires authentication, board membership, and owner role.
//	@Tags			boards
//	@Produce		json
//	@Param			boardId	path	string	true	"Board ID"
//	@Success		204		"Board deleted successfully"
//	@Failure		401		{object}	httputils.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httputils.ErrorResponse	"Insufficient permissions (not owner)"
//	@Failure		404		{object}	httputils.ErrorResponse	"Board not found"
//	@Failure		500		{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId} [delete]
func (h *BoardHandler) DeleteBoard(w http.ResponseWriter, r *http.Request) {
	user := middlewares.GetAuthenticatedUser(r.Context())
	boardID := middlewares.GetBoardID(r.Context())

	if err := h.boardService.DeleteBoard(r.Context(), boardID, user.ID); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusNoContent, nil)
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
//	@Failure		400		{object}	httputils.ErrorResponse	"Invalid request body or validation error"
//	@Failure		401		{object}	httputils.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httputils.ErrorResponse	"Not a member of this board or insufficient permissions"
//	@Failure		403		{object}	httputils.ErrorResponse	"Cannot modify owner's role"
//	@Failure		404		{object}	httputils.ErrorResponse	"Board member not found"
//	@Failure		500		{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/members/{userId}/role [patch]
func (h *BoardHandler) UpdateBoardMemberRole(w http.ResponseWriter, r *http.Request) {
	boardID := middlewares.GetBoardID(r.Context())
	userID := middlewares.GetUserID(r.Context())
	boardMemberRole := middlewares.GetBoardMemberRole(r.Context())

	var body dtos.UpdateBoardMemberRoleDto

	if err := httputils.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	if err := h.boardMemberService.UpdateBoardMemberRole(r.Context(), boardID, userID, boardMemberRole, body); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusNoContent, nil)
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
//	@Failure		401		{object}	httputils.ErrorResponse	"Unauthorized - missing or invalid authentication"
//	@Failure		403		{object}	httputils.ErrorResponse	"Not a member of this board or insufficient permissions"
//	@Failure		403		{object}	httputils.ErrorResponse	"Cannot remove board owner"
//	@Failure		404		{object}	httputils.ErrorResponse	"Board member not found"
//	@Failure		500		{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/boards/{boardId}/members/{userId} [delete]
func (h *BoardHandler) RemoveBoardMember(w http.ResponseWriter, r *http.Request) {
	boardID := middlewares.GetBoardID(r.Context())
	userID := middlewares.GetUserID(r.Context())
	boardMemberRole := middlewares.GetBoardMemberRole(r.Context())

	if err := h.boardMemberService.RemoveBoardMember(r.Context(), boardID, userID, boardMemberRole); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusNoContent, nil)
}
