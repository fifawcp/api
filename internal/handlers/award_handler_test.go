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

func newTestAwardHandler(as *mocks.MockAwardService) *AwardHandler {
	return NewAwardHandler(as, &mocks.MockLogger{}, validator.NewValidator())
}

// ---------------------------------------------------------------------------
// TestAwardHandler_GetMemberAwards
// ---------------------------------------------------------------------------
func TestAwardHandler_GetMemberAwards(t *testing.T) {
	t.Parallel()

	makeReq := func(t *testing.T, targetUserID string) *http.Request {
		t.Helper()
		return testutils.MakeJSONRequest(
			t, http.MethodGet, "/boards/1/members/"+targetUserID+"/awards", nil,
			testutils.WithUserID(targetUserID),
		)
	}

	t.Run("returns 200 with member awards on success", func(t *testing.T) {
		t.Parallel()

		targetUserID := gofakeit.UUID()

		as := &mocks.MockAwardService{
			GetMemberAwardsFunc: func(ctx context.Context, userID string) (*domain.UserAwards, error) {
				assert.Equal(t, targetUserID, userID)
				return &domain.UserAwards{IsLocked: true}, nil
			},
		}

		h := newTestAwardHandler(as)
		req := makeReq(t, targetUserID)
		w := httptest.NewRecorder()

		h.GetMemberAwards(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data domain.UserAwards `json:"data"`
		}
		testutils.ParseJSONResponse(t, w, &resp)
		assert.True(t, resp.Data.IsLocked)
	})

	t.Run("returns 403 when predictions are hidden", func(t *testing.T) {
		t.Parallel()

		targetUserID := gofakeit.UUID()

		as := &mocks.MockAwardService{
			GetMemberAwardsFunc: func(ctx context.Context, userID string) (*domain.UserAwards, error) {
				return nil, domain.ErrPredictionsHidden
			},
		}

		h := newTestAwardHandler(as)
		req := makeReq(t, targetUserID)
		w := httptest.NewRecorder()

		h.GetMemberAwards(w, req)

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

		as := &mocks.MockAwardService{
			GetMemberAwardsFunc: func(ctx context.Context, userID string) (*domain.UserAwards, error) {
				return nil, domain.ErrBoardMemberNotFound
			},
		}

		h := newTestAwardHandler(as)
		req := makeReq(t, targetUserID)
		w := httptest.NewRecorder()

		h.GetMemberAwards(w, req)

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
