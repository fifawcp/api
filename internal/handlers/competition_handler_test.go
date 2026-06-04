package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/fifawcp/api/internal/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCompetitionHandler(cs *mocks.MockCompetitionService) *CompetitionHandler {
	return NewCompetitionHandler(cs, testutils.NewTestConfig(), validator.NewValidator(), &mocks.MockLogger{})
}

func makeCreatePoolReq(t *testing.T, body any) *http.Request {
	return testutils.MakeJSONRequest(
		t, http.MethodPost, "/boards/1/competitions", body,
		testutils.WithAuthUser(testutils.CreateTestUser()),
		testutils.WithBoardID(1),
		testutils.WithBoardMemberRole(domain.BoardMemberRoleAdmin),
	)
}

func TestCompetitionHandler_CreateCompetition_Pool(t *testing.T) {
	t.Parallel()

	matchID := int64(42)

	t.Run("returns 201 on success", func(t *testing.T) {
		t.Parallel()

		cs := &mocks.MockCompetitionService{
			CreateCompetitionFunc: func(ctx context.Context, boardID int64, userID string, role domain.BoardMemberRole, payload dtos.CreateCompetitionDto) (*domain.CompetitionListItem, error) {
				return &domain.CompetitionListItem{
					Competition: domain.Competition{ID: 9, BoardID: boardID, Type: domain.CompetitionTypePool, Name: payload.Name, PoolMatchID: payload.MatchID},
				}, nil
			},
		}
		h := newTestCompetitionHandler(cs)

		req := makeCreatePoolReq(t, dtos.CreateCompetitionDto{Type: domain.CompetitionTypePool, Name: "CvP", MatchID: &matchID})
		w := httptest.NewRecorder()

		h.CreateCompetition(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data domain.CompetitionListItem `json:"data"`
		}
		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.CompetitionTypePool, resp.Data.Type)
		require.NotNil(t, resp.Data.PoolMatchID)
		assert.Equal(t, matchID, *resp.Data.PoolMatchID)
	})

	t.Run("returns 409 when a pool already exists for the match", func(t *testing.T) {
		t.Parallel()

		cs := &mocks.MockCompetitionService{
			CreateCompetitionFunc: func(ctx context.Context, boardID int64, userID string, role domain.BoardMemberRole, payload dtos.CreateCompetitionDto) (*domain.CompetitionListItem, error) {
				return nil, domain.ErrDuplicatePoolForMatch
			},
		}
		h := newTestCompetitionHandler(cs)

		req := makeCreatePoolReq(t, dtos.CreateCompetitionDto{Type: domain.CompetitionTypePool, Name: "CvP", MatchID: &matchID})
		w := httptest.NewRecorder()

		h.CreateCompetition(w, req)

		require.Equal(t, http.StatusConflict, w.Code)
		var resp httpx.ErrorResponse
		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "DUPLICATE_POOL_FOR_MATCH", resp.Error.Code)
	})
}
