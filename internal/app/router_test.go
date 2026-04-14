package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ncondes/fifawcp/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
)

func TestAppContainer_NewRouter(t *testing.T) {
	t.Parallel()

	app := newTestContainer(t, &config.Config{})

	router := app.NewRouter()

	t.Run("creates router with expected routes", func(t *testing.T) {
		t.Parallel()

		assert.NotNil(t, router)

		routes := router.Routes()

		assert.NotEmpty(t, routes)

	})

	t.Run("redirects / to /swagger/index.html", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMovedPermanently, w.Code)
		assert.Equal(t, "/swagger/index.html", w.Header().Get("Location"))
	})

	t.Run("redirects /swagger to /swagger/index.html", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/swagger", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMovedPermanently, w.Code)
		assert.Equal(t, "/swagger/index.html", w.Header().Get("Location"))
	})

	t.Run("returns 200 when making a request to /api/health", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
