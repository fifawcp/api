package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/test/testutils"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func newTestDebugHandler() *DebugHandler {
	cfg := &config.Config{}
	return NewDebugHandler(cfg)
}

// ---------------------------------------------------------------------------
// TestDebugHandler_RequestOtp
// ---------------------------------------------------------------------------

func TestDebugHandler_RequestOtp(t *testing.T) {
	t.Parallel()

	const identifier = "test@example.com"

	makeRequestOtpReq := func(t *testing.T) *http.Request {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/debug/auth/otp/request/"+identifier, nil)

		// chi normally injects URL params during routing; in tests we must
		// build the route context manually and attach it to the request.
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("identifier", identifier)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		return req
	}

	t.Run("returns 200 on success with OTP and expiration", func(t *testing.T) {
		t.Parallel()

		h := newTestDebugHandler()

		req := makeRequestOtpReq(t)

		w := httptest.NewRecorder()
		h.RequestTotp(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data map[string]any
		}

		testutils.ParseJSONResponse(t, w, &resp)

		assert.IsType(t, "", resp.Data["otp"])
		assert.NotEmpty(t, resp.Data["otp"])

		assert.IsType(t, float64(1), resp.Data["expires_in"])
		assert.NotEmpty(t, resp.Data["expires_in"])
	})
}
