package app

import (
	"database/sql"
	"errors"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/packages/mocks"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func newTestContainer(t *testing.T, cfg *config.Config) *Container {
	t.Helper()
	cfg.Server.ShutdownTimeout = 200 * time.Millisecond

	c := &Container{
		Config:        cfg,
		Logger:        &mocks.MockLogger{},
		Authenticator: &mocks.MockAuthenticator{},
		Scheduler:     &mocks.MockScheduler{},
		db:            &sql.DB{},
		redis:         &redis.Client{},
		validator:     &validator.Validator{},
		mailer:        &mocks.MockMailer{},
	}

	c.initRepositories()
	c.initStorages()
	c.initServices()
	c.initHandlers()
	c.initJobs()
	c.RateLimiters = newRateLimiters(c.redis, &cfg.RateLimit)
	c.shutdownServer = c.ShutdownServer

	return c
}

func TestAppContainer_NewAppContainer(t *testing.T) {
	t.Parallel()

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

	t.Run("creates container with all dependencies wired", func(t *testing.T) {
		t.Parallel()

		container := newTestContainer(t, cfg)

		assert.NotNil(t, container)
		assert.Equal(t, cfg, container.Config)
		assert.NotNil(t, container.Logger)
		assert.NotNil(t, container.AuthHandler)
		assert.NotNil(t, container.UserHandler)
		assert.NotNil(t, container.Authenticator)
		assert.NotNil(t, container.Scheduler)
		assert.NotNil(t, container.RateLimiters)
		assert.NotNil(t, container.shutdownServer)
		assert.Equal(t, 200*time.Millisecond, container.Config.Server.ShutdownTimeout)
	})

	t.Run("creates container even when job registration fails", func(t *testing.T) {
		t.Parallel()

		c := &Container{
			Config:        cfg,
			Logger:        &mocks.MockLogger{},
			Authenticator: &mocks.MockAuthenticator{},
			Scheduler: &mocks.MockScheduler{
				RegisterJobFunc: func(spec string, job domain.Job) error {
					return errors.New("job registration failed")
				},
			},
			db:        &sql.DB{},
			redis:     &redis.Client{},
			validator: &validator.Validator{},
			mailer:    &mocks.MockMailer{},
		}

		c.initRepositories()
		c.initStorages()
		c.initServices()
		c.initHandlers()
		c.initJobs()

		assert.NotNil(t, c)
	})
}

func TestAppContainer_StartServer(t *testing.T) {
	serverCfg := &config.Config{
		Port: "0",
		Server: config.ServerConfig{
			WriteTimeout: 1 * time.Second,
			ReadTimeout:  1 * time.Second,
			IdleTimeout:  1 * time.Second,
		},
	}

	t.Run("starts and shuts down gracefully on SIGTERM", func(t *testing.T) {
		container := newTestContainer(t, serverCfg)
		router := container.NewRouter()

		errChan := make(chan error, 1)
		go func() { errChan <- container.StartServer(router) }()

		time.Sleep(100 * time.Millisecond)
		process, _ := os.FindProcess(os.Getpid())
		process.Signal(syscall.SIGTERM)

		err := <-errChan
		assert.NoError(t, err)
	})

	t.Run("returns error when server fails to start", func(t *testing.T) {
		t.Parallel()

		container := newTestContainer(t, &config.Config{
			Port: "-1",
			Server: config.ServerConfig{
				WriteTimeout: 1 * time.Second,
				ReadTimeout:  1 * time.Second,
				IdleTimeout:  1 * time.Second,
			},
		})

		err := container.StartServer(container.NewRouter())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server error")
	})

	t.Run("returns error when graceful shutdown fails", func(t *testing.T) {
		container := newTestContainer(t, serverCfg)
		container.shutdownServer = func(server *http.Server) error {
			return errors.New("shutdown failed")
		}

		router := container.NewRouter()
		errChan := make(chan error, 1)
		go func() { errChan <- container.StartServer(router) }()

		time.Sleep(100 * time.Millisecond)
		process, _ := os.FindProcess(os.Getpid())
		process.Signal(syscall.SIGTERM)

		err := <-errChan
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shutdown failed")
	})
}

func TestAppContainer_ShutdownServer(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when server shuts down gracefully", func(t *testing.T) {
		t.Parallel()

		container := newTestContainer(t, &config.Config{})
		server := &http.Server{Addr: ":0"}

		err := container.ShutdownServer(server)
		assert.NoError(t, err)
	})

	t.Run("returns error when server shutdown times out", func(t *testing.T) {
		t.Parallel()

		container := newTestContainer(t, &config.Config{})

		listener, _ := net.Listen("tcp", ":0")
		server := &http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(10 * time.Second)
			}),
		}
		go server.Serve(listener) //nolint:errcheck

		go http.Get("http://" + listener.Addr().String() + "/") //nolint:errcheck
		time.Sleep(50 * time.Millisecond)

		// ShutdownTimeout is 200ms (set by newTestContainer), handler sleeps 10s
		// so context deadline hits first, triggering the error path
		err := container.ShutdownServer(server)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server shutdown failed")
	})
}
