package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/fifawcp/api/internal/test/testutils"
	"github.com/stretchr/testify/assert"
)

func newTestOAuthHandler(s *mocks.MockOAuthService) *OAuthHandler {
	return NewOAuthHandler(
		s,
		&mocks.MockLogger{},
		&config.Config{
			Auth: config.AuthConfig{
				GoogleOAuth: config.OAuthConfig{
					ClientID:          "test",
					ClientSecret:      "test",
					ReturnToAllowlist: []string{"http://localhost:3000"},
				},
			},
		},
	)
}

// ---------------------------------------------------------------------------
// TestOAuthHandler_GoogleOAuth
// ---------------------------------------------------------------------------
func TestOAuthHandler_GoogleOAuth(t *testing.T) {
	t.Parallel()

	mockGoogleOAuthReq := func(t *testing.T) *http.Request {
		t.Helper()

		return testutils.MakeJSONRequest(
			t, http.MethodGet, "/oauth/google?", nil,
		)
	}

	t.Run("returns 307 on success", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockOAuthService{
			BeginGoogleLoginFunc: func(context.Context, string) (string, error) { return "state", nil },
		}
		h := newTestOAuthHandler(s)

		req := mockGoogleOAuthReq(t)
		w := httptest.NewRecorder()

		h.GoogleOAuth(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockOAuthService{
			BeginGoogleLoginFunc: func(context.Context, string) (string, error) { return "", errors.New("internal server error") },
		}
		h := newTestOAuthHandler(s)

		req := mockGoogleOAuthReq(t)
		w := httptest.NewRecorder()
		h.GoogleOAuth(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

}

// ---------------------------------------------------------------------------
// TestOAuthHandler_GoogleOAuthCallback
// ---------------------------------------------------------------------------
func TestOAuthHandler_GoogleOAuthCallback(t *testing.T) {
	t.Parallel()

	mockGoogleOAuthCallbackReq := func(t *testing.T) *http.Request {
		t.Helper()

		return testutils.MakeJSONRequest(
			t, http.MethodGet, "/oauth/google/callback", nil,
			testutils.WithRequestInfo(&dtos.RequestInfo{
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
				DeviceInfo: dtos.DeviceInfo{
					Browser:     "test-browser",
					Platform:    "test-platform",
					OS:          "test-os",
					DisplayName: "test-device",
				},
			}),
		)
	}

	t.Run("returns 302 on success", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockOAuthService{
			CompleteGoogleLoginFunc: func(context.Context, string, string, dtos.RequestInfo) (*dtos.AuthenticationDto, string, error) {
				return &dtos.AuthenticationDto{}, "http://localhost:3000", nil
			},
		}
		h := newTestOAuthHandler(s)

		req := mockGoogleOAuthCallbackReq(t)
		w := httptest.NewRecorder()
		h.GoogleOAuthCallback(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
	})

	t.Run("propagates service error", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockOAuthService{
			CompleteGoogleLoginFunc: func(context.Context, string, string, dtos.RequestInfo) (*dtos.AuthenticationDto, string, error) {
				return nil, "", errors.New("internal server error")
			},
		}
		h := newTestOAuthHandler(s)

		req := mockGoogleOAuthCallbackReq(t)
		w := httptest.NewRecorder()
		h.GoogleOAuthCallback(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
