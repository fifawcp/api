package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/ncondes/fifawcp/internal/domain"
	"github.com/ncondes/fifawcp/internal/dtos"
	"github.com/ncondes/fifawcp/internal/infrastructure/config"
	"github.com/ncondes/fifawcp/internal/infrastructure/middlewares"
)

func NewTestConfig() *config.Config {
	return &config.Config{
		Env: "test",
	}
}

type RequestOption func(*http.Request) *http.Request

func MakeJSONRequest(
	t *testing.T,
	method string,
	url string,
	body any,
	options ...RequestOption,
) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("failed to encode request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")

	for _, option := range options {
		req = option(req)
	}

	return req
}

func WithRequestInfo(info *dtos.RequestInfo) RequestOption {
	return func(r *http.Request) *http.Request {
		ctx := context.WithValue(r.Context(), middlewares.RequestInfoContextKey, info)
		return r.WithContext(ctx)
	}
}

func WithAuthUser(user *domain.User) RequestOption {
	return func(r *http.Request) *http.Request {
		ctx := context.WithValue(r.Context(), middlewares.AuthenticatedUserContextKey, user)
		return r.WithContext(ctx)
	}
}

func WithBoardID(boardID string) RequestOption {
	return func(r *http.Request) *http.Request {
		ctx := context.WithValue(r.Context(), middlewares.BoardIDContextKey, boardID)
		return r.WithContext(ctx)
	}
}

func WithBoardMemberRole(role domain.BoardMemberRole) RequestOption {
	return func(r *http.Request) *http.Request {
		ctx := context.WithValue(r.Context(), middlewares.BoardMemberRoleContextKey, role)
		return r.WithContext(ctx)
	}
}

func WithUserID(userID string) RequestOption {
	return func(r *http.Request) *http.Request {
		ctx := context.WithValue(r.Context(), middlewares.UserIDContextKey, userID)
		return r.WithContext(ctx)
	}
}

func MakeRequestWithCookie(
	t *testing.T,
	method, url, cookieName, cookieValue string,
) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, url, nil)
	req.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: cookieValue,
	})
	return req
}

func ParseJSONResponse(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(w.Body).Decode(v); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
}

func GetResponseCookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func CreateTestUser() *domain.User {
	return &domain.User{
		ID:        gofakeit.UUID(),
		FirstName: gofakeit.FirstName(),
		LastName:  gofakeit.LastName(),
		Email:     gofakeit.Email(),
		Username:  gofakeit.Username(),
	}
}
