package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/fifawcp/api/internal/test/testutils"
	"github.com/stretchr/testify/assert"
)

func newTestUserHandler(s *mocks.MockUserService) *UserHandler {
	return NewUserHandler(
		s,
		&mocks.MockLogger{},
		validator.NewValidator(),
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

// ---------------------------------------------------------------------------
// TestUserHandler_UpdateProfile
// ---------------------------------------------------------------------------
func TestUserHandler_UpdateProfile(t *testing.T) {
	t.Parallel()

	authenticatedUser := testutils.CreateTestUser()

	t.Run("returns 200 with updated user on success", func(t *testing.T) {
		t.Parallel()

		newFirstName := "Updated"
		updatedUser := &domain.User{ID: authenticatedUser.ID, FirstName: newFirstName}

		s := &mocks.MockUserService{
			UpdateUserFunc: func(ctx context.Context, userID string, payload *dtos.UpdateUserDto) (*domain.User, error) {
				assert.Equal(t, authenticatedUser.ID, userID)
				assert.NotNil(t, payload.FirstName)
				assert.Equal(t, newFirstName, *payload.FirstName)
				return updatedUser, nil
			},
		}
		h := newTestUserHandler(s)

		req := testutils.MakeJSONRequest(
			t, http.MethodPatch, "/users/profile",
			dtos.UpdateUserDto{FirstName: &newFirstName},
			testutils.WithAuthUser(authenticatedUser),
		)
		w := httptest.NewRecorder()

		h.UpdateProfile(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data domain.User `json:"data"`
		}
		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, newFirstName, resp.Data.FirstName)
	})

	t.Run("returns 400 when no fields are provided", func(t *testing.T) {
		t.Parallel()

		h := newTestUserHandler(&mocks.MockUserService{})

		req := testutils.MakeJSONRequest(
			t, http.MethodPatch, "/users/profile",
			dtos.UpdateUserDto{},
			testutils.WithAuthUser(authenticatedUser),
		)
		w := httptest.NewRecorder()

		h.UpdateProfile(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
