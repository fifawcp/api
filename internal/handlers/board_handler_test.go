package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/ncondes/fifawcp/internal/domain"
	"github.com/ncondes/fifawcp/internal/dtos"
	"github.com/ncondes/fifawcp/internal/infrastructure/validator"
	"github.com/ncondes/fifawcp/internal/packages/testutils"
	"github.com/stretchr/testify/assert"
)

func newTestBoardHandler(
	bs *testutils.MockBoardService,
	bms *testutils.MockBoardMemberService,
	brs *testutils.MockBoardRankingService,
) *BoardHandler {
	return NewBoardHandler(
		bs,
		bms,
		brs,
		testutils.NewTestConfig(),
		validator.NewValidator(),
		&testutils.MockLogger{},
	)
}

// ---------------------------------------------------------------------------
// TestBoardHandler_CreateBoard
// ---------------------------------------------------------------------------
func TestBoardHandler_CreateBoard(t *testing.T) {
	t.Parallel()

	makeCreateBoardReq := func(t *testing.T, body any) *http.Request {
		t.Helper()

		user := testutils.CreateTestUser()

		req := testutils.MakeJSONRequest(
			t, http.MethodPost, "/boards", body,
			testutils.WithAuthUser(user),
		)

		return req
	}

	t.Run("returns 201 on success with board data", func(t *testing.T) {
		t.Parallel()

		bs := &testutils.MockBoardService{
			CreateBoardFunc: func(
				ctx context.Context,
				payload dtos.CreateBoardDto,
				userID string,
			) (*domain.Board, error) {
				return &domain.Board{
					ID:   "test-board-id",
					Name: payload.Name,
				}, nil
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		body := dtos.CreateBoardDto{
			Name: "Test Board",
		}

		req := makeCreateBoardReq(t, body)
		w := httptest.NewRecorder()

		h.CreateBoard(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Data domain.Board `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)

		assert.Equal(t, "test-board-id", resp.Data.ID)
		assert.Equal(t, "Test Board", resp.Data.Name)
	})

	t.Run("returns 400 on validation fails", func(t *testing.T) {
		t.Parallel()

		h := newTestBoardHandler(nil, nil, nil)

		testCases := []struct {
			name        string
			payload     dtos.CreateBoardDto
			expectedKey string
			expectedMsg string
		}{
			{
				name:        "missing name",
				payload:     dtos.CreateBoardDto{},
				expectedKey: "name",
				expectedMsg: "name is required",
			},
			{
				name:        "name too long",
				payload:     dtos.CreateBoardDto{Name: strings.Repeat("a", 121)},
				expectedKey: "name",
				expectedMsg: "name must be at most 120 characters",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := makeCreateBoardReq(t, tc.payload)
				w := httptest.NewRecorder()

				h.CreateBoard(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)

				var resp struct {
					Error   string            `json:"error"`
					Details map[string]string `json:"details"`
				}

				testutils.ParseJSONResponse(t, w, &resp)
				assert.Equal(t, "validation failed", resp.Error)
				assert.Equal(t, tc.expectedMsg, resp.Details[tc.expectedKey])
			})
		}
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bs := &testutils.MockBoardService{
			CreateBoardFunc: func(
				ctx context.Context,
				payload dtos.CreateBoardDto,
				userID string,
			) (*domain.Board, error) {
				return nil, errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		body := dtos.CreateBoardDto{
			Name: "Test Board",
		}

		req := makeCreateBoardReq(t, body)
		w := httptest.NewRecorder()

		h.CreateBoard(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestBoardHandler_GetUserBoards
// ---------------------------------------------------------------------------
func TestBoardHandler_GetUserBoards(t *testing.T) {
	t.Parallel()

	makeGetUserBoardsReq := func(t *testing.T) *http.Request {
		t.Helper()

		user := testutils.CreateTestUser()

		return testutils.MakeJSONRequest(
			t, http.MethodGet, "/boards", nil,
			testutils.WithAuthUser(user),
		)
	}

	t.Run("returns 200 with user boards", func(t *testing.T) {
		t.Parallel()

		boards := []*domain.Board{
			{ID: "board-1", Name: "Test Board 1"},
			{ID: "board-2", Name: "Test Board 2"},
		}

		bs := &testutils.MockBoardService{
			GetUserBoardsFunc: func(
				ctx context.Context,
				userID string,
			) ([]*domain.Board, error) {
				return boards, nil
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		req := makeGetUserBoardsReq(t)
		w := httptest.NewRecorder()

		h.GetUserBoards(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []domain.Board `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Len(t, resp.Data, 2)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bs := &testutils.MockBoardService{
			GetUserBoardsFunc: func(
				ctx context.Context,
				userID string,
			) ([]*domain.Board, error) {
				return nil, errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		req := makeGetUserBoardsReq(t)
		w := httptest.NewRecorder()

		h.GetUserBoards(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestBoardHandler_JoinBoard
// ---------------------------------------------------------------------------
func TestBoardHandler_JoinBoard(t *testing.T) {
	t.Parallel()

	makeJoinBoardReq := func(t *testing.T, body dtos.JoinBoardDto) *http.Request {
		t.Helper()

		user := testutils.CreateTestUser()

		return testutils.MakeJSONRequest(
			t, http.MethodPost, "/boards/join", body,
			testutils.WithAuthUser(user),
		)
	}

	t.Run("return 204 when join is successful", func(t *testing.T) {
		t.Parallel()

		body := dtos.JoinBoardDto{
			JoinCode: "ABCD1234",
		}

		bms := &testutils.MockBoardMemberService{
			JoinBoardFunc: func(ctx context.Context, joinCode string, userID string) error {
				return nil
			},
		}

		h := newTestBoardHandler(nil, bms, nil)

		req := makeJoinBoardReq(t, body)
		w := httptest.NewRecorder()

		h.JoinBoard(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns 400 on validation fails", func(t *testing.T) {
		t.Parallel()

		h := newTestBoardHandler(nil, nil, nil)

		testCases := []struct {
			name        string
			payload     dtos.JoinBoardDto
			expectedKey string
			expectedMsg string
		}{
			{
				name:        "missing join code",
				payload:     dtos.JoinBoardDto{},
				expectedKey: "join_code",
				expectedMsg: "join_code is required",
			},
			{
				name:        "join code too long",
				payload:     dtos.JoinBoardDto{JoinCode: strings.Repeat("A", 9)},
				expectedKey: "join_code",
				expectedMsg: "join_code must be at most 8 characters",
			},
			{
				name:        "join code too short",
				payload:     dtos.JoinBoardDto{JoinCode: "ABCD123"},
				expectedKey: "join_code",
				expectedMsg: "join_code must be at least 8 characters",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := makeJoinBoardReq(t, tc.payload)
				w := httptest.NewRecorder()

				h.JoinBoard(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)

				var resp struct {
					Error   string            `json:"error"`
					Details map[string]string `json:"details"`
				}

				testutils.ParseJSONResponse(t, w, &resp)
				assert.Equal(t, "validation failed", resp.Error)
				assert.Equal(t, tc.expectedMsg, resp.Details[tc.expectedKey])
			})
		}
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		body := dtos.JoinBoardDto{
			JoinCode: "ABCD1234",
		}

		bms := &testutils.MockBoardMemberService{
			JoinBoardFunc: func(ctx context.Context, joinCode string, userID string) error {
				return errors.New("db error")
			},
		}

		h := newTestBoardHandler(nil, bms, nil)

		req := makeJoinBoardReq(t, body)
		w := httptest.NewRecorder()

		h.JoinBoard(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

}

// ---------------------------------------------------------------------------
// TestBoardHandler_GetBoardByID
// ---------------------------------------------------------------------------
func TestBoardHandler_GetBoardByID(t *testing.T) {
	t.Parallel()

	makeGetBoardByIDReq := func(t *testing.T) *http.Request {
		t.Helper()

		boardID := gofakeit.UUID()
		req := testutils.MakeJSONRequest(
			t, http.MethodGet, "/boards/"+boardID, nil,
			testutils.WithBoardID(boardID),
		)

		return req
	}

	t.Run("returns 200 with board data", func(t *testing.T) {
		t.Parallel()

		bs := &testutils.MockBoardService{
			GetBoardByIDFunc: func(ctx context.Context, boardID string) (*domain.Board, error) {
				return &domain.Board{
					ID:   boardID,
					Name: "Test Board",
				}, nil
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		req := makeGetBoardByIDReq(t)
		w := httptest.NewRecorder()

		h.GetBoardByID(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data *domain.Board `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "Test Board", resp.Data.Name)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bs := &testutils.MockBoardService{
			GetBoardByIDFunc: func(ctx context.Context, boardID string) (*domain.Board, error) {
				return nil, errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		req := makeGetBoardByIDReq(t)
		w := httptest.NewRecorder()

		h.GetBoardByID(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestBoardHandler_GetBoardMembers
// ---------------------------------------------------------------------------
func TestBoardHandler_GetBoardMembers(t *testing.T) {
	t.Parallel()

	makeGetBoardMembersReq := func(t *testing.T) *http.Request {
		t.Helper()

		boardID := gofakeit.UUID()
		req := testutils.MakeJSONRequest(
			t, http.MethodGet, "/boards/"+boardID+"/members", nil,
			testutils.WithBoardID(boardID),
		)

		return req
	}

	t.Run("returns 200 with board members data", func(t *testing.T) {
		t.Parallel()

		bms := &testutils.MockBoardMemberService{
			GetBoardMembersFunc: func(ctx context.Context, boardID string) ([]*domain.BoardMember, error) {
				return []*domain.BoardMember{
					{
						BoardID: boardID,
						UserID:  gofakeit.UUID(),
						Role:    domain.BoardMemberRoleAdmin,
					},
				}, nil
			},
		}

		h := newTestBoardHandler(nil, bms, nil)

		req := makeGetBoardMembersReq(t)
		w := httptest.NewRecorder()

		h.GetBoardMembers(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []*domain.BoardMember `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Len(t, resp.Data, 1)
		assert.Equal(t, domain.BoardMemberRoleAdmin, resp.Data[0].Role)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bms := &testutils.MockBoardMemberService{
			GetBoardMembersFunc: func(ctx context.Context, boardID string) ([]*domain.BoardMember, error) {
				return nil, errors.New("db error")
			},
		}

		h := newTestBoardHandler(nil, bms, nil)

		req := makeGetBoardMembersReq(t)
		w := httptest.NewRecorder()

		h.GetBoardMembers(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestBoardHandler_RegenerateJoinCode
// ---------------------------------------------------------------------------
func TestBoardHandler_RegenerateJoinCode(t *testing.T) {
	t.Parallel()

	makeRegenerateJoinCodeReq := func(t *testing.T) *http.Request {
		t.Helper()

		boardID := gofakeit.UUID()
		req := testutils.MakeJSONRequest(
			t, http.MethodPost, "/boards/"+boardID+"/regenerate-join-code", nil,
			testutils.WithBoardID(boardID),
		)

		return req
	}

	t.Run("returns 200 with new join code", func(t *testing.T) {
		t.Parallel()

		newJoinCode := "NEWCODE1"

		bs := &testutils.MockBoardService{
			RegenerateJoinCodeFunc: func(ctx context.Context, boardID string) (string, error) {
				return newJoinCode, nil
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		req := makeRegenerateJoinCodeReq(t)
		w := httptest.NewRecorder()

		h.RegenerateJoinCode(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data dtos.JoinBoardDto `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, newJoinCode, resp.Data.JoinCode)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bs := &testutils.MockBoardService{
			RegenerateJoinCodeFunc: func(ctx context.Context, boardID string) (string, error) {
				return "", errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		req := makeRegenerateJoinCodeReq(t)
		w := httptest.NewRecorder()

		h.RegenerateJoinCode(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestBoardHandler_UpdateBoard
// ---------------------------------------------------------------------------
func TestBoardHandler_UpdateBoard(t *testing.T) {
	t.Parallel()

	makeUpdateBoardReq := func(t *testing.T, body any) *http.Request {
		t.Helper()

		boardID := gofakeit.UUID()
		req := testutils.MakeJSONRequest(
			t, http.MethodPatch, "/boards/"+boardID, body,
			testutils.WithBoardID(boardID),
			testutils.WithBoardMemberRole(domain.BoardMemberRoleAdmin),
		)

		return req
	}

	t.Run("returns 204 on success", func(t *testing.T) {
		t.Parallel()

		bs := &testutils.MockBoardService{
			UpdateBoardFunc: func(
				ctx context.Context,
				boardID string,
				boardMemberRole domain.BoardMemberRole,
				body dtos.UpdateBoardDto,
			) error {
				return nil
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		body := dtos.UpdateBoardDto{
			Name: "Updated Board Name",
		}

		req := makeUpdateBoardReq(t, body)
		w := httptest.NewRecorder()

		h.UpdateBoard(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns 400 on validation fails", func(t *testing.T) {
		t.Parallel()

		h := newTestBoardHandler(nil, nil, nil)

		testCases := []struct {
			name        string
			payload     dtos.UpdateBoardDto
			expectedKey string
			expectedMsg string
		}{
			{
				name:        "missing name",
				payload:     dtos.UpdateBoardDto{},
				expectedKey: "name",
				expectedMsg: "name is required",
			},
			{
				name:        "name too long",
				payload:     dtos.UpdateBoardDto{Name: strings.Repeat("a", 121)},
				expectedKey: "name",
				expectedMsg: "name must be at most 120 characters",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := makeUpdateBoardReq(t, tc.payload)
				w := httptest.NewRecorder()

				h.UpdateBoard(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)

				var resp struct {
					Error   string            `json:"error"`
					Details map[string]string `json:"details"`
				}

				testutils.ParseJSONResponse(t, w, &resp)
				assert.Equal(t, "validation failed", resp.Error)
				assert.Equal(t, tc.expectedMsg, resp.Details[tc.expectedKey])
			})
		}
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bs := &testutils.MockBoardService{
			UpdateBoardFunc: func(
				ctx context.Context,
				boardID string,
				boardMemberRole domain.BoardMemberRole,
				body dtos.UpdateBoardDto,
			) error {
				return errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		body := dtos.UpdateBoardDto{
			Name: "Updated Board Name",
		}

		req := makeUpdateBoardReq(t, body)
		w := httptest.NewRecorder()

		h.UpdateBoard(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestBoardHandler_DeleteBoard
// ---------------------------------------------------------------------------
func TestBoardHandler_DeleteBoard(t *testing.T) {
	t.Parallel()

	makeDeleteBoardReq := func(t *testing.T) *http.Request {
		t.Helper()

		user := testutils.CreateTestUser()
		boardID := gofakeit.UUID()
		req := testutils.MakeJSONRequest(
			t, http.MethodDelete, "/boards/"+boardID, nil,
			testutils.WithAuthUser(user),
			testutils.WithBoardID(boardID),
		)

		return req
	}

	t.Run("returns 204 on success", func(t *testing.T) {
		t.Parallel()

		bs := &testutils.MockBoardService{
			DeleteBoardFunc: func(ctx context.Context, boardID string, userID string) error {
				return nil
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		req := makeDeleteBoardReq(t)
		w := httptest.NewRecorder()

		h.DeleteBoard(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bs := &testutils.MockBoardService{
			DeleteBoardFunc: func(ctx context.Context, boardID string, userID string) error {
				return errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil, nil)

		req := makeDeleteBoardReq(t)
		w := httptest.NewRecorder()

		h.DeleteBoard(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestBoardHandler_UpdateBoardMemberRole
// ---------------------------------------------------------------------------
func TestBoardHandler_UpdateBoardMemberRole(t *testing.T) {
	t.Parallel()

	makeUpdateBoardMemberRoleReq := func(t *testing.T, body any) *http.Request {
		t.Helper()

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()
		req := testutils.MakeJSONRequest(
			t, http.MethodPatch, "/boards/"+boardID+"/members/"+userID+"/role", body,
			testutils.WithBoardID(boardID),
			testutils.WithUserID(userID),
			testutils.WithBoardMemberRole(domain.BoardMemberRoleAdmin),
		)

		return req
	}

	t.Run("returns 204 on success", func(t *testing.T) {
		t.Parallel()

		bms := &testutils.MockBoardMemberService{
			UpdateBoardMemberRoleFunc: func(
				ctx context.Context,
				boardID string,
				userID string,
				boardMemberRole domain.BoardMemberRole,
				body dtos.UpdateBoardMemberRoleDto,
			) error {
				return nil
			},
		}

		h := newTestBoardHandler(nil, bms, nil)

		body := dtos.UpdateBoardMemberRoleDto{
			Role: domain.BoardMemberRoleAdmin,
		}

		req := makeUpdateBoardMemberRoleReq(t, body)
		w := httptest.NewRecorder()

		h.UpdateBoardMemberRole(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns 400 on validation fails", func(t *testing.T) {
		t.Parallel()

		h := newTestBoardHandler(nil, nil, nil)

		testCases := []struct {
			name        string
			payload     dtos.UpdateBoardMemberRoleDto
			expectedKey string
			expectedMsg string
		}{
			{
				name:        "missing role",
				payload:     dtos.UpdateBoardMemberRoleDto{},
				expectedKey: "role",
				expectedMsg: "role is required",
			},
			{
				name:        "invalid role",
				payload:     dtos.UpdateBoardMemberRoleDto{Role: "invalid"},
				expectedKey: "role",
				expectedMsg: "role is invalid",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := makeUpdateBoardMemberRoleReq(t, tc.payload)
				w := httptest.NewRecorder()

				h.UpdateBoardMemberRole(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)

				var resp struct {
					Error   string            `json:"error"`
					Details map[string]string `json:"details"`
				}

				testutils.ParseJSONResponse(t, w, &resp)
				assert.Equal(t, "validation failed", resp.Error)
				assert.Equal(t, tc.expectedMsg, resp.Details[tc.expectedKey])
			})
		}
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bms := &testutils.MockBoardMemberService{
			UpdateBoardMemberRoleFunc: func(
				ctx context.Context,
				boardID string,
				userID string,
				boardMemberRole domain.BoardMemberRole,
				body dtos.UpdateBoardMemberRoleDto,
			) error {
				return errors.New("db error")
			},
		}

		h := newTestBoardHandler(nil, bms, nil)

		body := dtos.UpdateBoardMemberRoleDto{
			Role: domain.BoardMemberRoleAdmin,
		}

		req := makeUpdateBoardMemberRoleReq(t, body)
		w := httptest.NewRecorder()

		h.UpdateBoardMemberRole(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestBoardHandler_RemoveBoardMember
// ---------------------------------------------------------------------------
func TestBoardHandler_RemoveBoardMember(t *testing.T) {
	t.Parallel()

	makeRemoveBoardMemberReq := func(t *testing.T) *http.Request {
		t.Helper()

		boardID := gofakeit.UUID()
		userID := gofakeit.UUID()
		req := testutils.MakeJSONRequest(
			t, http.MethodDelete, "/boards/"+boardID+"/members/"+userID, nil,
			testutils.WithBoardID(boardID),
			testutils.WithUserID(userID),
			testutils.WithBoardMemberRole(domain.BoardMemberRoleAdmin),
		)

		return req
	}

	t.Run("returns 204 on success", func(t *testing.T) {
		t.Parallel()

		bms := &testutils.MockBoardMemberService{
			RemoveBoardMemberFunc: func(
				ctx context.Context,
				boardID string,
				userID string,
				boardMemberRole domain.BoardMemberRole,
			) error {
				return nil
			},
		}

		h := newTestBoardHandler(nil, bms, nil)

		req := makeRemoveBoardMemberReq(t)
		w := httptest.NewRecorder()

		h.RemoveBoardMember(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bms := &testutils.MockBoardMemberService{
			RemoveBoardMemberFunc: func(
				ctx context.Context,
				boardID string,
				userID string,
				boardMemberRole domain.BoardMemberRole,
			) error {
				return errors.New("db error")
			},
		}

		h := newTestBoardHandler(nil, bms, nil)

		req := makeRemoveBoardMemberReq(t)
		w := httptest.NewRecorder()

		h.RemoveBoardMember(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestBoardHandler_GetBoardRanking
// ---------------------------------------------------------------------------
func TestBoardHandler_GetBoardRanking(t *testing.T) {
	t.Parallel()

	makeGetBoardRankingReq := func(t *testing.T) *http.Request {
		t.Helper()

		boardID := gofakeit.UUID()
		req := testutils.MakeJSONRequest(
			t, http.MethodGet, "/boards/"+boardID+"/ranking", nil,
			testutils.WithBoardID(boardID),
		)

		return req
	}

	t.Run("returns 200 with board ranking data", func(t *testing.T) {
		t.Parallel()

		brs := &testutils.MockBoardRankingService{
			GetBoardRankingFunc: func(ctx context.Context, boardID string) ([]*domain.BoardRanking, error) {
				return []*domain.BoardRanking{
					{
						BoardID:         gofakeit.UUID(),
						UserID:          gofakeit.UUID(),
						TotalPoints:     gofakeit.Number(1, 100),
						GlobalPoints:    gofakeit.Number(1, 100),
						DetailedPoints:  gofakeit.Number(1, 100),
						ExactHits:       gofakeit.Number(1, 100),
						CorrectOutcomes: gofakeit.Number(1, 100),
						UpdatedAt:       gofakeit.Date().Format(time.RFC3339),
					},
				}, nil
			},
		}

		h := newTestBoardHandler(nil, nil, brs)

		req := makeGetBoardRankingReq(t)
		w := httptest.NewRecorder()

		h.GetBoardRanking(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []*domain.BoardRanking `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Len(t, resp.Data, 1)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		brs := &testutils.MockBoardRankingService{
			GetBoardRankingFunc: func(ctx context.Context, boardID string) ([]*domain.BoardRanking, error) {
				return nil, errors.New("db error")
			},
		}

		h := newTestBoardHandler(nil, nil, brs)

		req := makeGetBoardRankingReq(t)
		w := httptest.NewRecorder()

		h.GetBoardRanking(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
