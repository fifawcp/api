package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ncondes/fifawcp/internal/domain"
	"github.com/ncondes/fifawcp/internal/packages/mocks"
	"github.com/ncondes/fifawcp/internal/packages/testutils"
	"github.com/stretchr/testify/assert"
)

func newTestUserHandler(s *mocks.MockUserService) *UserHandler {
	return NewUserHandler(
		s,
		&mocks.MockLogger{},
	)
}

// ---------------------------------------------------------------------------
// TestUserHandler_GetProfile
// ---------------------------------------------------------------------------
func TestUserHandler_GetProfile(t *testing.T) {
	t.Parallel()

	makeGetProfileReq := func(t *testing.T, user *domain.User) *http.Request {
		t.Helper()

		req := testutils.MakeJSONRequest(
			t, http.MethodGet, "/user/profile", nil,
			testutils.WithAuthUser(user),
		)

		return req
	}

	authenticatedUser := testutils.CreateTestUser()

	t.Run("returns 200 with user profile on success", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockUserService{
			GetUserFunc: func(
				ctx context.Context,
				userID string,
			) (*domain.User, error) {
				return authenticatedUser, nil
			},
		}
		h := newTestUserHandler(s)

		req := makeGetProfileReq(t, authenticatedUser)
		w := httptest.NewRecorder()

		h.GetProfile(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data domain.User `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)

		assert.Equal(t, authenticatedUser.ID, resp.Data.ID)
		assert.Equal(t, authenticatedUser.FirstName, resp.Data.FirstName)
		assert.Equal(t, authenticatedUser.LastName, resp.Data.LastName)
		assert.Equal(t, authenticatedUser.Email, resp.Data.Email)
		assert.Equal(t, authenticatedUser.Username, resp.Data.Username)
	})
}
