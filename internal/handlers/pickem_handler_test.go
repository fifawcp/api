package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/fifawcp/api/internal/test/testutils"
	"github.com/stretchr/testify/assert"
)

func newTestPickemHandler(ps *mocks.MockPickemService) *PickemHandler {
	return NewPickemHandler(ps, &mocks.MockLogger{}, validator.NewValidator())
}

// ---------------------------------------------------------------------------
// TestPickemHandler_GetMemberPickem
// ---------------------------------------------------------------------------
func TestPickemHandler_GetMemberPickem(t *testing.T) {
	t.Parallel()

	makeReq := func(t *testing.T, targetUserID string) *http.Request {
		t.Helper()
		return testutils.MakeJSONRequest(
			t, http.MethodGet, "/boards/1/members/"+targetUserID+"/pickem", nil,
			testutils.WithUserID(targetUserID),
		)
	}

	t.Run("returns 200 with member pickem on success", func(t *testing.T) {
		t.Parallel()

		targetUserID := gofakeit.UUID()

		ps := &mocks.MockPickemService{
			GetMemberPickemFunc: func(ctx context.Context, userID string) (*domain.UserPickem, error) {
				assert.Equal(t, targetUserID, userID)
				return &domain.UserPickem{IsLocked: true}, nil
			},
		}

		h := newTestPickemHandler(ps)
		req := makeReq(t, targetUserID)
		w := httptest.NewRecorder()

		h.GetMemberPickem(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data domain.UserPickem `json:"data"`
		}
		testutils.ParseJSONResponse(t, w, &resp)
		assert.True(t, resp.Data.IsLocked)
	})

	t.Run("returns 403 when predictions are hidden", func(t *testing.T) {
		t.Parallel()

		targetUserID := gofakeit.UUID()

		ps := &mocks.MockPickemService{
			GetMemberPickemFunc: func(ctx context.Context, userID string) (*domain.UserPickem, error) {
				return nil, domain.ErrPredictionsHidden
			},
		}

		h := newTestPickemHandler(ps)
		req := makeReq(t, targetUserID)
		w := httptest.NewRecorder()

		h.GetMemberPickem(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "PREDICTIONS_HIDDEN", resp.Error.Code)
	})

	t.Run("returns 404 when board member not found", func(t *testing.T) {
		t.Parallel()

		targetUserID := gofakeit.UUID()

		ps := &mocks.MockPickemService{
			GetMemberPickemFunc: func(ctx context.Context, userID string) (*domain.UserPickem, error) {
				return nil, domain.ErrBoardMemberNotFound
			},
		}

		h := newTestPickemHandler(ps)
		req := makeReq(t, targetUserID)
		w := httptest.NewRecorder()

		h.GetMemberPickem(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "BOARD_MEMBER_NOT_FOUND", resp.Error.Code)
	})
}
