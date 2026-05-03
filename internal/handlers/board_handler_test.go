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
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/fifawcp/api/internal/test/testutils"
	"github.com/stretchr/testify/assert"
)

func newTestBoardHandler(
	bs *mocks.MockBoardService,
	bms *mocks.MockBoardMemberService,
) *BoardHandler {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			SessionTTL: 24 * time.Hour,
		},
	}
	return NewBoardHandler(
		bs,
		bms,
		cfg,
		validator.NewValidator(),
		&mocks.MockLogger{},
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

		bs := &mocks.MockBoardService{
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

		h := newTestBoardHandler(bs, nil)

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

		h := newTestBoardHandler(nil, nil)

		testCases := []struct {
			name         string
			payload      dtos.CreateBoardDto
			expectedKey  string
			expectedCode string
		}{
			{
				name:         "missing name",
				payload:      dtos.CreateBoardDto{},
				expectedKey:  "name",
				expectedCode: "REQUIRED",
			},
			{
				name:         "name too long",
				payload:      dtos.CreateBoardDto{Name: strings.Repeat("a", 121)},
				expectedKey:  "name",
				expectedCode: "MAX_LENGTH",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				req := makeCreateBoardReq(t, tc.payload)
				w := httptest.NewRecorder()

				h.CreateBoard(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)

				var resp struct {
					Error struct {
						Code   string `json:"code"`
						Fields map[string]struct {
							Code string `json:"code"`
						} `json:"fields"`
					} `json:"error"`
				}

				testutils.ParseJSONResponse(t, w, &resp)
				assert.Equal(t, "VALIDATION_FAILED", resp.Error.Code)
				assert.Equal(t, tc.expectedCode, resp.Error.Fields[tc.expectedKey].Code)
			})
		}
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bs := &mocks.MockBoardService{
			CreateBoardFunc: func(
				ctx context.Context,
				payload dtos.CreateBoardDto,
				userID string,
			) (*domain.Board, error) {
				return nil, errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil)

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

		boards := []*domain.BoardSummary{
			{ID: "board-1", Name: "Test Board 1", UserRank: 1, MembersCount: 2},
			{ID: "board-2", Name: "Test Board 2", UserRank: 2, MembersCount: 3},
		}

		bs := &mocks.MockBoardService{
			GetUserBoardsFunc: func(
				ctx context.Context,
				userID string,
			) ([]*domain.BoardSummary, error) {
				return boards, nil
			},
		}

		h := newTestBoardHandler(bs, nil)

		req := makeGetUserBoardsReq(t)
		w := httptest.NewRecorder()

		h.GetUserBoards(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []domain.BoardSummary `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Len(t, resp.Data, 2)
		assert.Equal(t, 2, resp.Data[0].MembersCount)
		assert.Equal(t, 1, resp.Data[0].UserRank)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bs := &mocks.MockBoardService{
			GetUserBoardsFunc: func(
				ctx context.Context,
				userID string,
			) ([]*domain.BoardSummary, error) {
				return nil, errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil)

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

		bms := &mocks.MockBoardMemberService{
			JoinBoardFunc: func(ctx context.Context, joinCode string, userID string) error {
				return nil
			},
		}

		h := newTestBoardHandler(nil, bms)

		req := makeJoinBoardReq(t, body)
		w := httptest.NewRecorder()

		h.JoinBoard(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns 400 on validation fails", func(t *testing.T) {
		t.Parallel()

		h := newTestBoardHandler(nil, nil)

		testCases := []struct {
			name         string
			payload      dtos.JoinBoardDto
			expectedKey  string
			expectedCode string
		}{
			{
				name:         "missing join code",
				payload:      dtos.JoinBoardDto{},
				expectedKey:  "join_code",
				expectedCode: "REQUIRED",
			},
			{
				name:         "join code too long",
				payload:      dtos.JoinBoardDto{JoinCode: strings.Repeat("A", 9)},
				expectedKey:  "join_code",
				expectedCode: "MAX_LENGTH",
			},
			{
				name:         "join code too short",
				payload:      dtos.JoinBoardDto{JoinCode: "ABCD123"},
				expectedKey:  "join_code",
				expectedCode: "MIN_LENGTH",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				req := makeJoinBoardReq(t, tc.payload)
				w := httptest.NewRecorder()

				h.JoinBoard(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)

				var resp struct {
					Error struct {
						Code   string `json:"code"`
						Fields map[string]struct {
							Code string `json:"code"`
						} `json:"fields"`
					} `json:"error"`
				}

				testutils.ParseJSONResponse(t, w, &resp)
				assert.Equal(t, "VALIDATION_FAILED", resp.Error.Code)
				assert.Equal(t, tc.expectedCode, resp.Error.Fields[tc.expectedKey].Code)
			})
		}
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		body := dtos.JoinBoardDto{
			JoinCode: "ABCD1234",
		}

		bms := &mocks.MockBoardMemberService{
			JoinBoardFunc: func(ctx context.Context, joinCode string, userID string) error {
				return errors.New("db error")
			},
		}

		h := newTestBoardHandler(nil, bms)

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

		bs := &mocks.MockBoardService{
			GetBoardByIDFunc: func(ctx context.Context, boardID string) (*domain.BoardDetails, error) {
				return &domain.BoardDetails{
					ID:       boardID,
					Name:     "Test Board",
					JoinCode: "ABCD1234",
					Members: []*domain.BoardDetailsMember{
						{
							UserID:      gofakeit.UUID(),
							UserName:    "johndoe",
							Role:        domain.BoardMemberRoleMember,
							JoinedAt:    time.Now(),
							Rank:        1,
							TotalPoints: 10,
						},
					},
				}, nil
			},
		}

		h := newTestBoardHandler(bs, nil)

		req := makeGetBoardByIDReq(t)
		w := httptest.NewRecorder()

		h.GetBoardByID(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data *domain.BoardDetails `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "Test Board", resp.Data.Name)
		assert.Equal(t, "ABCD1234", resp.Data.JoinCode)
		assert.Len(t, resp.Data.Members, 1)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bs := &mocks.MockBoardService{
			GetBoardByIDFunc: func(ctx context.Context, boardID string) (*domain.BoardDetails, error) {
				return nil, errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil)

		req := makeGetBoardByIDReq(t)
		w := httptest.NewRecorder()

		h.GetBoardByID(w, req)

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

		bs := &mocks.MockBoardService{
			RegenerateJoinCodeFunc: func(ctx context.Context, boardID string) (string, error) {
				return newJoinCode, nil
			},
		}

		h := newTestBoardHandler(bs, nil)

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

		bs := &mocks.MockBoardService{
			RegenerateJoinCodeFunc: func(ctx context.Context, boardID string) (string, error) {
				return "", errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil)

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

		bs := &mocks.MockBoardService{
			UpdateBoardFunc: func(
				ctx context.Context,
				boardID string,
				boardMemberRole domain.BoardMemberRole,
				body dtos.UpdateBoardDto,
			) error {
				return nil
			},
		}

		h := newTestBoardHandler(bs, nil)

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

		h := newTestBoardHandler(nil, nil)

		testCases := []struct {
			name         string
			payload      dtos.UpdateBoardDto
			expectedKey  string
			expectedCode string
		}{
			{
				name:         "missing name",
				payload:      dtos.UpdateBoardDto{},
				expectedKey:  "name",
				expectedCode: "REQUIRED",
			},
			{
				name:         "name too long",
				payload:      dtos.UpdateBoardDto{Name: strings.Repeat("a", 121)},
				expectedKey:  "name",
				expectedCode: "MAX_LENGTH",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				req := makeUpdateBoardReq(t, tc.payload)
				w := httptest.NewRecorder()

				h.UpdateBoard(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)

				var resp struct {
					Error struct {
						Code   string `json:"code"`
						Fields map[string]struct {
							Code string `json:"code"`
						} `json:"fields"`
					} `json:"error"`
				}

				testutils.ParseJSONResponse(t, w, &resp)
				assert.Equal(t, "VALIDATION_FAILED", resp.Error.Code)
				assert.Equal(t, tc.expectedCode, resp.Error.Fields[tc.expectedKey].Code)
			})
		}
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bs := &mocks.MockBoardService{
			UpdateBoardFunc: func(
				ctx context.Context,
				boardID string,
				boardMemberRole domain.BoardMemberRole,
				body dtos.UpdateBoardDto,
			) error {
				return errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil)

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

		bs := &mocks.MockBoardService{
			DeleteBoardFunc: func(ctx context.Context, boardID string, userID string) error {
				return nil
			},
		}

		h := newTestBoardHandler(bs, nil)

		req := makeDeleteBoardReq(t)
		w := httptest.NewRecorder()

		h.DeleteBoard(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bs := &mocks.MockBoardService{
			DeleteBoardFunc: func(ctx context.Context, boardID string, userID string) error {
				return errors.New("db error")
			},
		}

		h := newTestBoardHandler(bs, nil)

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

		bms := &mocks.MockBoardMemberService{
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

		h := newTestBoardHandler(nil, bms)

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

		h := newTestBoardHandler(nil, nil)

		testCases := []struct {
			name         string
			payload      dtos.UpdateBoardMemberRoleDto
			expectedKey  string
			expectedCode string
		}{
			{
				name:         "missing role",
				payload:      dtos.UpdateBoardMemberRoleDto{},
				expectedKey:  "role",
				expectedCode: "REQUIRED",
			},
			{
				name:         "invalid role",
				payload:      dtos.UpdateBoardMemberRoleDto{Role: "invalid"},
				expectedKey:  "role",
				expectedCode: "INVALID_OPTION",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				req := makeUpdateBoardMemberRoleReq(t, tc.payload)
				w := httptest.NewRecorder()

				h.UpdateBoardMemberRole(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)

				var resp struct {
					Error struct {
						Code   string `json:"code"`
						Fields map[string]struct {
							Code string `json:"code"`
						} `json:"fields"`
					} `json:"error"`
				}

				testutils.ParseJSONResponse(t, w, &resp)
				assert.Equal(t, "VALIDATION_FAILED", resp.Error.Code)
				assert.Equal(t, tc.expectedCode, resp.Error.Fields[tc.expectedKey].Code)
			})
		}
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bms := &mocks.MockBoardMemberService{
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

		h := newTestBoardHandler(nil, bms)

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

		bms := &mocks.MockBoardMemberService{
			RemoveBoardMemberFunc: func(
				ctx context.Context,
				boardID string,
				userID string,
				boardMemberRole domain.BoardMemberRole,
			) error {
				return nil
			},
		}

		h := newTestBoardHandler(nil, bms)

		req := makeRemoveBoardMemberReq(t)
		w := httptest.NewRecorder()

		h.RemoveBoardMember(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bms := &mocks.MockBoardMemberService{
			RemoveBoardMemberFunc: func(
				ctx context.Context,
				boardID string,
				userID string,
				boardMemberRole domain.BoardMemberRole,
			) error {
				return errors.New("db error")
			},
		}

		h := newTestBoardHandler(nil, bms)

		req := makeRemoveBoardMemberReq(t)
		w := httptest.NewRecorder()

		h.RemoveBoardMember(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestBoardHandler_LeaveBoard
// ---------------------------------------------------------------------------
func TestBoardHandler_LeaveBoard(t *testing.T) {
	t.Parallel()

	makeLeaveBoardReq := func(t *testing.T) *http.Request {
		t.Helper()

		boardID := gofakeit.UUID()
		user := testutils.CreateTestUser()
		req := testutils.MakeJSONRequest(
			t, http.MethodDelete, "/boards/"+boardID+"/leave", nil,
			testutils.WithAuthUser(user),
			testutils.WithBoardID(boardID),
		)

		return req
	}

	t.Run("returns 204 on success", func(t *testing.T) {
		t.Parallel()

		bms := &mocks.MockBoardMemberService{
			LeaveBoardFunc: func(ctx context.Context, boardID string, userID string) error {
				assert.NotEmpty(t, boardID)
				assert.NotEmpty(t, userID)
				return nil
			},
		}

		h := newTestBoardHandler(nil, bms)

		req := makeLeaveBoardReq(t)
		w := httptest.NewRecorder()

		h.LeaveBoard(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		bms := &mocks.MockBoardMemberService{
			LeaveBoardFunc: func(ctx context.Context, boardID string, userID string) error {
				return domain.ErrForbidden
			},
		}

		h := newTestBoardHandler(nil, bms)

		req := makeLeaveBoardReq(t)
		w := httptest.NewRecorder()

		h.LeaveBoard(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
