package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/fifawcp/api/internal/test/testutils"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func newTestAuthHandler(s *mocks.MockAuthService) *AuthHandler {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			SessionTTL:     24 * time.Hour,
			OTPTTL:         10 * time.Minute,
			OTPCooldown:    30 * time.Second,
			MaxOTPAttempts: 5,
		},
		Server: config.ServerConfig{
			ShutdownTimeout: 200 * time.Millisecond,
		},
	}
	return NewAuthHandler(
		s,
		&mocks.MockLogger{},
		validator.NewValidator(),
		cfg,
	)
}

// ---------------------------------------------------------------------------
// TestAuthHandler_RequestOtp
// ---------------------------------------------------------------------------

func TestAuthHandler_RequestOtp(t *testing.T) {
	t.Parallel()

	loginPurpose := domain.OTPPurposeLogin
	registrationPurpose := domain.OTPPurposeRegistration

	makeRequestOtpReq := func(t *testing.T, body any) *http.Request {
		t.Helper()

		return testutils.MakeJSONRequest(
			t, http.MethodPost, "/auth/otp/request", body,
		)
	}

	t.Run("returns 204 on success (login)", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			RequestOtpFunc: func(context.Context, *dtos.RequestOtpDto) error { return nil },
		}
		h := newTestAuthHandler(s)

		req := makeRequestOtpReq(t,
			dtos.RequestOtpDto{
				Identifier: "john@example.com",
				Purpose:    &loginPurpose,
			},
		)
		w := httptest.NewRecorder()
		h.RequestOtp(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns 204 on success (registration)", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			RequestOtpFunc: func(context.Context, *dtos.RequestOtpDto) error { return nil },
		}
		h := newTestAuthHandler(s)

		req := makeRequestOtpReq(t,
			dtos.RequestOtpDto{
				Identifier: "john@example.com",
				Purpose:    &registrationPurpose,
			},
		)
		w := httptest.NewRecorder()
		h.RequestOtp(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns 400 when validation fails", func(t *testing.T) {
		t.Parallel()

		h := newTestAuthHandler(&mocks.MockAuthService{})

		testCases := []struct {
			name        string
			payload     map[string]any
			expectedKey string
			expectedMsg string
		}{
			{
				name:        "missing identifier",
				payload:     map[string]any{"purpose": "login"},
				expectedKey: "identifier",
				expectedMsg: "identifier is required",
			},
			{
				name:        "missing purpose",
				payload:     map[string]any{"identifier": "john@example.com"},
				expectedKey: "purpose",
				expectedMsg: "purpose is required",
			},
			{
				name:        "invalid purpose value",
				payload:     map[string]any{"identifier": "john@example.com", "purpose": "unknown"},
				expectedKey: "purpose",
				expectedMsg: "purpose is invalid",
			},
			{
				name:        "identifier too long",
				payload:     map[string]any{"identifier": strings.Repeat("a", 256), "purpose": "login"},
				expectedKey: "identifier",
				expectedMsg: "identifier must be at most 255 characters",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				req := makeRequestOtpReq(t, tc.payload)
				w := httptest.NewRecorder()
				h.RequestOtp(w, req)

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

	t.Run("returns 409 when user already exists (registration)", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			RequestOtpFunc: func(context.Context, *dtos.RequestOtpDto) error {
				return domain.ErrUserAlreadyExists
			},
		}
		h := newTestAuthHandler(s)

		req := makeRequestOtpReq(t, dtos.RequestOtpDto{
			Identifier: "john@example.com",
			Purpose:    &registrationPurpose,
		})
		w := httptest.NewRecorder()
		h.RequestOtp(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrUserAlreadyExists.Error(), resp.Error)
	})

	t.Run("returns 401 when user already exists (login)", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			RequestOtpFunc: func(context.Context, *dtos.RequestOtpDto) error {
				return domain.ErrInvalidCredentials
			},
		}
		h := newTestAuthHandler(s)

		req := makeRequestOtpReq(t, dtos.RequestOtpDto{
			Identifier: "john@example.com",
			Purpose:    &loginPurpose,
		})
		w := httptest.NewRecorder()
		h.RequestOtp(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrInvalidCredentials.Error(), resp.Error)
	})

	t.Run("returns 429 when OTP cooldown is active", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			RequestOtpFunc: func(context.Context, *dtos.RequestOtpDto) error {
				return domain.ErrOtpCooldown(30 * time.Second)
			},
		}
		h := newTestAuthHandler(s)

		req := makeRequestOtpReq(t, dtos.RequestOtpDto{
			Identifier: "john@example.com",
			Purpose:    &loginPurpose,
		})
		w := httptest.NewRecorder()
		h.RequestOtp(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrOtpCooldown(30*time.Second).Error(), resp.Error)
	})

	t.Run("does not write a response when request context is cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // already cancelled

		s := &mocks.MockAuthService{
			RequestOtpFunc: func(context.Context, *dtos.RequestOtpDto) error {
				return context.Canceled
			},
		}
		h := newTestAuthHandler(s)

		req := makeRequestOtpReq(t, map[string]any{"identifier": "john@example.com", "purpose": "login"})
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		h.RequestOtp(w, req)

		assert.Empty(t, w.Body.String()) // no response written
	})

	t.Run("returns 500 on internal server error", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			RequestOtpFunc: func(context.Context, *dtos.RequestOtpDto) error {
				return errors.New("database connection error")
			},
		}
		h := newTestAuthHandler(s)

		req := makeRequestOtpReq(t, dtos.RequestOtpDto{
			Identifier: "john@example.com",
			Purpose:    &loginPurpose,
		})
		w := httptest.NewRecorder()
		h.RequestOtp(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "internal server error", resp.Error)
	})

	t.Run("does not write a response when request context is cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		s := &mocks.MockAuthService{
			RequestOtpFunc: func(context.Context, *dtos.RequestOtpDto) error {
				return context.Canceled
			},
		}
		h := newTestAuthHandler(s)

		req := makeRequestOtpReq(t, map[string]any{"identifier": "john@example.com", "purpose": "login"})
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		h.RequestOtp(w, req)

		assert.Empty(t, w.Body.String())
	})
}

// ---------------------------------------------------------------------------
// TestAuthHandler_Authenticate
// ---------------------------------------------------------------------------

func TestAuthHandler_Authenticate(t *testing.T) {
	t.Parallel()

	defaultRequestInfo := &dtos.RequestInfo{
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		DeviceInfo: dtos.DeviceInfo{
			Browser:     "test-browser",
			Platform:    "test-platform",
			OS:          "test-os",
			DisplayName: "test-device",
		},
	}

	makeAuthReq := func(t *testing.T, body any) *http.Request {
		t.Helper()

		req := testutils.MakeJSONRequest(
			t, http.MethodPost, "/auth/token", body,
			testutils.WithRequestInfo(defaultRequestInfo),
		)

		return req
	}

	t.Run("returns 200 with access token and sets refresh token cookie (login)", func(t *testing.T) {
		t.Parallel()

		expectedAccessToken := "access-token-value"
		expectedRefreshToken := "refresh-token-value"
		expectedExpiresAt := time.Now().Add(7 * 24 * time.Hour)
		expectedUser := testutils.CreateTestUser()

		s := &mocks.MockAuthService{
			AuthenticateFunc: func(
				context.Context,
				*dtos.AuthenticationInputDto,
				dtos.RequestInfo,
			) (*dtos.AuthenticationDto, error) {
				return &dtos.AuthenticationDto{
					User: expectedUser,
					Auth: dtos.AuthData{
						AccessToken:  expectedAccessToken,
						RefreshToken: expectedRefreshToken,
						ExpiresAt:    expectedExpiresAt,
					},
				}, nil
			},
		}
		h := newTestAuthHandler(s)

		req := makeAuthReq(t, map[string]any{
			"identifier": "john@example.com",
			"purpose":    "login",
			"otp":        "123456",
		})
		w := httptest.NewRecorder()

		h.Authenticate(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data dtos.AuthenticationDto `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)

		assert.Equal(t, expectedAccessToken, resp.Data.Auth.AccessToken)
		assert.NotContains(t, w.Body.String(), expectedRefreshToken, "refresh token must not appear in response body")
		assert.WithinDuration(t, expectedExpiresAt, resp.Data.Auth.ExpiresAt, time.Second)
		assert.Equal(t, expectedUser.ID, resp.Data.User.ID)
		assert.Equal(t, expectedUser.Email, resp.Data.User.Email)

		cookie := testutils.GetResponseCookie(w, "refresh_token")

		assert.NotNil(t, cookie, "refresh_token cookie should be set")
		assert.Equal(t, expectedRefreshToken, cookie.Value)
		assert.True(t, cookie.HttpOnly)
	})

	t.Run("returns 200 with access token and sets refresh token cookie (registration)", func(t *testing.T) {
		t.Parallel()

		expectedAccessToken := "access-token-value"
		expectedRefreshToken := "refresh-token-value"
		expectedExpiresAt := time.Now().Add(time.Hour)
		expectedUser := testutils.CreateTestUser()

		s := &mocks.MockAuthService{
			AuthenticateFunc: func(
				context.Context,
				*dtos.AuthenticationInputDto,
				dtos.RequestInfo,
			) (*dtos.AuthenticationDto, error) {
				return &dtos.AuthenticationDto{
					User: expectedUser,
					Auth: dtos.AuthData{
						AccessToken:  expectedAccessToken,
						RefreshToken: expectedRefreshToken,
						ExpiresAt:    expectedExpiresAt,
					},
				}, nil
			},
		}
		h := newTestAuthHandler(s)

		req := makeAuthReq(t, map[string]any{
			"identifier": expectedUser.Email,
			"purpose":    "registration",
			"otp":        "123456",
			"user": map[string]any{
				"email":      expectedUser.Email,
				"username":   expectedUser.Username,
				"first_name": expectedUser.FirstName,
				"last_name":  expectedUser.LastName,
			},
		})
		w := httptest.NewRecorder()

		h.Authenticate(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data dtos.AuthenticationDto `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)

		assert.Equal(t, expectedAccessToken, resp.Data.Auth.AccessToken)
		assert.NotContains(t, w.Body.String(), expectedRefreshToken, "refresh token must not appear in response body")
		assert.WithinDuration(t, expectedExpiresAt, resp.Data.Auth.ExpiresAt, 5*time.Second)

		assert.Equal(t, expectedUser.Username, resp.Data.User.Username)
		assert.Equal(t, expectedUser.Email, resp.Data.User.Email)
		assert.Equal(t, expectedUser.FirstName, resp.Data.User.FirstName)
		assert.Equal(t, expectedUser.LastName, resp.Data.User.LastName)

		cookie := testutils.GetResponseCookie(w, "refresh_token")

		assert.NotNil(t, cookie, "refresh_token cookie should be set")
		assert.Equal(t, expectedRefreshToken, cookie.Value)
		assert.True(t, cookie.HttpOnly)
	})

	t.Run("returns 400 when validation fails", func(t *testing.T) {
		t.Parallel()

		h := newTestAuthHandler(&mocks.MockAuthService{})

		testCases := []struct {
			name        string
			payload     map[string]any
			expectedKey string
			expectedMsg string
		}{
			{
				name:        "missing identifier",
				payload:     map[string]any{"purpose": "login", "otp": "123456"},
				expectedKey: "identifier",
				expectedMsg: "identifier is required",
			},
			{
				name:        "identifier too long",
				payload:     map[string]any{"identifier": strings.Repeat("a", 256), "purpose": "login"},
				expectedKey: "identifier",
				expectedMsg: "identifier must be at most 255 characters",
			},
			{
				name:        "missing otp",
				payload:     map[string]any{"identifier": "john@example.com", "purpose": "login"},
				expectedKey: "otp",
				expectedMsg: "otp is required",
			},
			{
				name:        "otp too short",
				payload:     map[string]any{"identifier": "john@example.com", "purpose": "login", "otp": "12345"},
				expectedKey: "otp",
				expectedMsg: "otp must be at least 6 characters",
			},
			{
				name:        "otp too long",
				payload:     map[string]any{"identifier": "john@example.com", "purpose": "login", "otp": "1234567"},
				expectedKey: "otp",
				expectedMsg: "otp must be at most 6 characters",
			},
			{
				name:        "invalid purpose",
				payload:     map[string]any{"identifier": "john@example.com", "purpose": "unknown", "otp": "123456"},
				expectedKey: "purpose",
				expectedMsg: "purpose is invalid",
			},

			{
				name: "registration without user field",
				payload: map[string]any{
					"identifier": "john@example.com",
					"purpose":    "registration",
					"otp":        "123456",
				},
				expectedKey: "User",
				expectedMsg: "User is required",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				req := makeAuthReq(t, tc.payload)
				w := httptest.NewRecorder()

				h.Authenticate(w, req)

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

	t.Run("returns 401 when OTP is invalid or expired", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			AuthenticateFunc: func(
				context.Context,
				*dtos.AuthenticationInputDto,
				dtos.RequestInfo,
			) (*dtos.AuthenticationDto, error) {
				return nil, domain.ErrOTPInvalidOrExpired
			},
		}
		h := newTestAuthHandler(s)

		req := makeAuthReq(t, map[string]any{
			"identifier": "john@example.com",
			"purpose":    "login",
			"otp":        "000000",
		})
		w := httptest.NewRecorder()

		h.Authenticate(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrOTPInvalidOrExpired.Error(), resp.Error)
	})

	t.Run("returns 401 when invalid credentials", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			AuthenticateFunc: func(
				context.Context,
				*dtos.AuthenticationInputDto,
				dtos.RequestInfo,
			) (*dtos.AuthenticationDto, error) {
				return nil, domain.ErrInvalidCredentials
			},
		}
		h := newTestAuthHandler(s)

		req := makeAuthReq(t, map[string]any{
			"identifier": "john@example.com",
			"purpose":    "login",
			"otp":        "123456",
		})
		w := httptest.NewRecorder()

		h.Authenticate(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrInvalidCredentials.Error(), resp.Error)
	})

	t.Run("returns 409 when user already exists (registration)", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			AuthenticateFunc: func(
				context.Context,
				*dtos.AuthenticationInputDto,
				dtos.RequestInfo,
			) (*dtos.AuthenticationDto, error) {
				return nil, domain.ErrUserAlreadyExists
			},
		}
		h := newTestAuthHandler(s)

		req := makeAuthReq(t, map[string]any{
			"identifier": "john@example.com",
			"purpose":    "registration",
			"otp":        "123456",
			"user": map[string]any{
				"email":      "john@example.com",
				"username":   "johndoe",
				"first_name": "John",
				"last_name":  "Doe",
			},
		})
		w := httptest.NewRecorder()

		h.Authenticate(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrUserAlreadyExists.Error(), resp.Error)
	})

	t.Run("returns 409 when username already taken (registration)", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			AuthenticateFunc: func(
				context.Context,
				*dtos.AuthenticationInputDto,
				dtos.RequestInfo,
			) (*dtos.AuthenticationDto, error) {
				return nil, domain.ErrUsernameAlreadyExists
			},
		}
		h := newTestAuthHandler(s)

		req := makeAuthReq(t, map[string]any{
			"identifier": "john@example.com",
			"purpose":    "registration",
			"otp":        "123456",
			"user": map[string]any{
				"email":      "john@example.com",
				"username":   "taken-username",
				"first_name": "John",
				"last_name":  "Doe",
			},
		})
		w := httptest.NewRecorder()

		h.Authenticate(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrUsernameAlreadyExists.Error(), resp.Error)
	})

	t.Run("returns 429 when too many OTP attempts (login)", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			AuthenticateFunc: func(
				context.Context,
				*dtos.AuthenticationInputDto,
				dtos.RequestInfo,
			) (*dtos.AuthenticationDto, error) {
				return nil, domain.ErrOTPTooManyAttempts
			},
		}
		h := newTestAuthHandler(s)

		req := makeAuthReq(t, map[string]any{
			"identifier": "john@example.com",
			"purpose":    "login",
			"otp":        "000000",
		})
		w := httptest.NewRecorder()

		h.Authenticate(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrOTPTooManyAttempts.Error(), resp.Error)
	})

	t.Run("returns 429 when too many OTP attempts (registration)", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			AuthenticateFunc: func(
				context.Context,
				*dtos.AuthenticationInputDto,
				dtos.RequestInfo,
			) (*dtos.AuthenticationDto, error) {
				return nil, domain.ErrOTPTooManyAttempts
			},
		}
		h := newTestAuthHandler(s)

		req := makeAuthReq(t, map[string]any{
			"identifier": "john@example.com",
			"purpose":    "registration",
			"otp":        "000000",
			"user": map[string]any{
				"email":      "john@example.com",
				"username":   "taken-username",
				"first_name": "John",
				"last_name":  "Doe",
			},
		})
		w := httptest.NewRecorder()

		h.Authenticate(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrOTPTooManyAttempts.Error(), resp.Error)
	})

	t.Run("returns 500 on internal server error", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			AuthenticateFunc: func(
				context.Context,
				*dtos.AuthenticationInputDto,
				dtos.RequestInfo,
			) (*dtos.AuthenticationDto, error) {
				return nil, errors.New("unexpected db failure")
			},
		}
		h := newTestAuthHandler(s)

		req := makeAuthReq(t, map[string]any{
			"identifier": "john@example.com",
			"purpose":    "login",
			"otp":        "123456",
		})
		w := httptest.NewRecorder()

		h.Authenticate(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "internal server error", resp.Error)
	})
}

// ---------------------------------------------------------------------------
// TestAuthHandler_RefreshToken
// ---------------------------------------------------------------------------

func TestAuthHandler_RefreshToken(t *testing.T) {
	t.Parallel()

	const cookieName = "refresh_token"
	const existingToken = "existing-refresh-token"

	makeRefreshTokenReq := func(t *testing.T, cookieValue string) *http.Request {
		t.Helper()

		return testutils.MakeRequestWithCookie(t, http.MethodPost, "/auth/token/refresh", cookieName, cookieValue)
	}

	t.Run("returns 200 and rotates the refresh token cookie", func(t *testing.T) {
		t.Parallel()

		newToken := "new-refresh-token"
		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		s := &mocks.MockAuthService{
			RefreshTokenFunc: func(context.Context, string) (*dtos.AuthData, error) {
				return &dtos.AuthData{
					AccessToken:  "new-access-token",
					RefreshToken: newToken,
					ExpiresAt:    expiresAt,
				}, nil
			},
		}
		h := newTestAuthHandler(s)

		req := makeRefreshTokenReq(t, existingToken)
		w := httptest.NewRecorder()

		h.RefreshToken(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		cookie := testutils.GetResponseCookie(w, cookieName)

		assert.NotNil(t, cookie)
		assert.Equal(t, newToken, cookie.Value)
		assert.True(t, cookie.HttpOnly)

		var resp struct {
			Data dtos.AuthData `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "new-access-token", resp.Data.AccessToken)
		assert.WithinDuration(t, expiresAt, resp.Data.ExpiresAt, time.Second)
	})

	t.Run("returns 401 when refresh token cookie is missing", func(t *testing.T) {
		t.Parallel()

		h := newTestAuthHandler(&mocks.MockAuthService{})

		req := httptest.NewRequest(http.MethodPost, "/auth/token/refresh", nil)
		w := httptest.NewRecorder()

		h.RefreshToken(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "missing refresh token", resp.Error)
	})

	t.Run("returns 401 when refresh token is invalid or expired", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			RefreshTokenFunc: func(context.Context, string) (*dtos.AuthData, error) {
				return nil, domain.ErrRefreshTokenInvalidOrExpired
			},
		}
		h := newTestAuthHandler(s)

		req := makeRefreshTokenReq(t, existingToken)
		w := httptest.NewRecorder()

		h.RefreshToken(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrRefreshTokenInvalidOrExpired.Error(), resp.Error)
	})

	t.Run("returns 500 on internal server error", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			RefreshTokenFunc: func(context.Context, string) (*dtos.AuthData, error) {
				return nil, errors.New("db error")
			},
		}
		h := newTestAuthHandler(s)

		req := makeRefreshTokenReq(t, existingToken)
		w := httptest.NewRecorder()

		h.RefreshToken(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestAuthHandler_Logout
// ---------------------------------------------------------------------------

func TestAuthHandler_Logout(t *testing.T) {
	t.Parallel()

	const cookieName = "refresh_token"
	const existingToken = "valid-refresh-token"

	makeLogoutReq := func(t *testing.T, cookieValue string) *http.Request {
		t.Helper()

		return testutils.MakeRequestWithCookie(t, http.MethodPost, "/auth/logout", cookieName, cookieValue)
	}

	t.Run("returns 204 and clears the refresh token cookie", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			LogoutFunc: func(context.Context, string) error { return nil },
		}
		h := newTestAuthHandler(s)

		req := makeLogoutReq(t, existingToken)
		w := httptest.NewRecorder()

		h.Logout(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		cookie := testutils.GetResponseCookie(w, cookieName)

		assert.NotNil(t, cookie)
		assert.Equal(t, "", cookie.Value)
		assert.Equal(t, -1, cookie.MaxAge)
	})

	t.Run("returns 401 when refresh token cookie is missing", func(t *testing.T) {
		t.Parallel()

		h := newTestAuthHandler(&mocks.MockAuthService{})

		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		w := httptest.NewRecorder()

		h.Logout(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "missing refresh token", resp.Error)
	})

	t.Run("returns 401 when refresh token is invalid or expired", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			LogoutFunc: func(context.Context, string) error {
				return domain.ErrRefreshTokenInvalidOrExpired
			},
		}
		h := newTestAuthHandler(s)

		req := makeLogoutReq(t, existingToken)
		w := httptest.NewRecorder()

		h.Logout(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrRefreshTokenInvalidOrExpired.Error(), resp.Error)
	})

	t.Run("returns 500 on internal server error", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			LogoutFunc: func(context.Context, string) error {
				return errors.New("db error")
			},
		}
		h := newTestAuthHandler(s)

		req := makeLogoutReq(t, existingToken)
		w := httptest.NewRecorder()

		h.Logout(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// TestAuthHandler_LogoutAll
// ---------------------------------------------------------------------------

func TestAuthHandler_LogoutAll(t *testing.T) {
	t.Parallel()

	const cookieName = "refresh_token"
	const existingToken = "valid-refresh-token"

	makeLogoutAllReq := func(t *testing.T, cookieValue string) *http.Request {
		t.Helper()

		return testutils.MakeRequestWithCookie(t, http.MethodPost, "/auth/logout/all", cookieName, cookieValue)
	}

	t.Run("returns 204 and clears the refresh token cookie", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			LogoutAllFunc: func(context.Context, string) error { return nil },
		}
		h := newTestAuthHandler(s)

		req := makeLogoutAllReq(t, existingToken)
		w := httptest.NewRecorder()

		h.LogoutAll(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		cookie := testutils.GetResponseCookie(w, cookieName)

		assert.NotNil(t, cookie)
		assert.Equal(t, "", cookie.Value)
		assert.Equal(t, -1, cookie.MaxAge)
	})

	t.Run("returns 401 when refresh token cookie is missing", func(t *testing.T) {
		t.Parallel()

		h := newTestAuthHandler(&mocks.MockAuthService{})

		req := httptest.NewRequest(http.MethodPost, "/auth/logout/all", nil)
		w := httptest.NewRecorder()

		h.LogoutAll(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "missing refresh token", resp.Error)
	})

	t.Run("returns 401 when refresh token is invalid or expired", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			LogoutAllFunc: func(context.Context, string) error {
				return domain.ErrRefreshTokenInvalidOrExpired
			},
		}
		h := newTestAuthHandler(s)

		req := makeLogoutAllReq(t, existingToken)
		w := httptest.NewRecorder()

		h.LogoutAll(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrRefreshTokenInvalidOrExpired.Error(), resp.Error)
	})

	t.Run("returns 500 on internal server error", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			LogoutAllFunc: func(context.Context, string) error {
				return errors.New("db error")
			},
		}
		h := newTestAuthHandler(s)

		req := makeLogoutAllReq(t, existingToken)
		w := httptest.NewRecorder()

		h.LogoutAll(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "internal server error", resp.Error)
	})
}

// ---------------------------------------------------------------------------
// TestAuthHandler_GetSessions
// ---------------------------------------------------------------------------

func TestAuthHandler_GetSessions(t *testing.T) {
	t.Parallel()

	const cookieName = "refresh_token"
	const existingToken = "valid-refresh-token"

	makeGetSessionsReq := func(t *testing.T, cookieValue string) *http.Request {
		t.Helper()

		return testutils.MakeRequestWithCookie(t, http.MethodGet, "/auth/sessions", cookieName, cookieValue)
	}

	t.Run("returns 200 with a list of sessions", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		expectedSessions := []domain.Session{
			{
				ID:        "session-1",
				UserID:    "user-123",
				IPAddress: "127.0.0.1",
				CreatedAt: now,
			},
			{
				ID:        "session-2",
				UserID:    "user-123",
				IPAddress: "10.0.0.1",
				CreatedAt: now,
			},
		}

		s := &mocks.MockAuthService{
			GetSessionsFunc: func(context.Context, string) ([]domain.Session, error) {
				return expectedSessions, nil
			},
		}
		h := newTestAuthHandler(s)

		req := makeGetSessionsReq(t, existingToken)
		w := httptest.NewRecorder()

		h.GetSessions(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []domain.Session `json:"data"`
		}

		testutils.ParseJSONResponse(t, w, &resp)

		assert.Len(t, resp.Data, 2)
		for _, session := range resp.Data {
			assert.IsType(t, "", session.ID)
			assert.NotEmpty(t, session.ID)

			assert.IsType(t, "", session.UserID)
			assert.NotEmpty(t, session.UserID)

			assert.IsType(t, "", session.IPAddress)
			assert.NotEmpty(t, session.IPAddress)

			assert.IsType(t, time.Time{}, session.CreatedAt)
			assert.NotEmpty(t, session.CreatedAt)
		}
	})

	t.Run("returns 401 when refresh token cookie is missing", func(t *testing.T) {
		t.Parallel()

		h := newTestAuthHandler(&mocks.MockAuthService{})

		req := httptest.NewRequest(http.MethodGet, "/auth/sessions", nil)
		w := httptest.NewRecorder()

		h.GetSessions(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error string `json:"error"`
		}
		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "missing refresh token", resp.Error)
	})

	t.Run("returns 401 when refresh token is invalid or expired", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			GetSessionsFunc: func(context.Context, string) ([]domain.Session, error) {
				return nil, domain.ErrRefreshTokenInvalidOrExpired
			},
		}
		h := newTestAuthHandler(s)

		req := makeGetSessionsReq(t, existingToken)
		w := httptest.NewRecorder()

		h.GetSessions(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrRefreshTokenInvalidOrExpired.Error(), resp.Error)
	})

	t.Run("returns 500 on internal server error", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			GetSessionsFunc: func(context.Context, string) ([]domain.Session, error) {
				return nil, errors.New("db error")
			},
		}
		h := newTestAuthHandler(s)

		req := makeGetSessionsReq(t, existingToken)
		w := httptest.NewRecorder()

		h.GetSessions(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "internal server error", resp.Error)
	})
}

// ---------------------------------------------------------------------------
// TestAuthHandler_DeleteSession
// ---------------------------------------------------------------------------

func TestAuthHandler_DeleteSession(t *testing.T) {
	t.Parallel()

	const sessionID = "session-uuid-123"

	makeDeleteReq := func(t *testing.T, user *domain.User) *http.Request {
		t.Helper()

		req := testutils.MakeJSONRequest(
			t, http.MethodDelete, "/auth/sessions/"+sessionID, nil,
			testutils.WithAuthUser(user),
		)

		// chi normally injects URL params during routing; in tests we must
		// build the route context manually and attach it to the request.
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", sessionID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		return req
	}

	authenticatedUser := testutils.CreateTestUser()

	t.Run("returns 204 on success", func(t *testing.T) {
		t.Parallel()

		var capturedSessionID, capturedUserID string

		s := &mocks.MockAuthService{
			DeleteSessionFunc: func(
				ctx context.Context,
				sessionID string,
				userID string,
			) error {
				capturedSessionID = sessionID
				capturedUserID = userID
				return nil
			},
		}
		h := newTestAuthHandler(s)

		req := makeDeleteReq(t, authenticatedUser)
		w := httptest.NewRecorder()

		h.DeleteSession(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, sessionID, capturedSessionID)
		assert.Equal(t, authenticatedUser.ID, capturedUserID)
	})

	t.Run("returns 404 when session is not found", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			DeleteSessionFunc: func(
				ctx context.Context,
				sessionID string,
				userID string,
			) error {
				return domain.ErrSessionNotFound
			},
		}
		h := newTestAuthHandler(s)

		req := makeDeleteReq(t, authenticatedUser)
		w := httptest.NewRecorder()

		h.DeleteSession(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, domain.ErrSessionNotFound.Error(), resp.Error)
	})

	t.Run("returns 500 on internal server error", func(t *testing.T) {
		t.Parallel()

		s := &mocks.MockAuthService{
			DeleteSessionFunc: func(
				ctx context.Context,
				sessionID string,
				userID string,
			) error {
				return errors.New("db error")
			},
		}
		h := newTestAuthHandler(s)

		req := makeDeleteReq(t, authenticatedUser)
		w := httptest.NewRecorder()

		h.DeleteSession(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp struct {
			Error string `json:"error"`
		}

		testutils.ParseJSONResponse(t, w, &resp)
		assert.Equal(t, "internal server error", resp.Error)
	})
}
