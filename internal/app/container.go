package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ncondes/fifa-world-cup-pickems/internal/handlers"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/auth"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/config"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/logging"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/middlewares"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/validator"
	"github.com/ncondes/fifa-world-cup-pickems/internal/repositories"
	"github.com/ncondes/fifa-world-cup-pickems/internal/services"
	"github.com/redis/go-redis/v9"
)

type AppContainer struct {
	Config        *config.Config
	Logger        logging.Logger
	AuthHandler   *handlers.AuthHandler
	Authenticator auth.Authenticator
}

func NewAppContainer(
	cfg *config.Config,
	logger logging.Logger,
	db *sql.DB,
	redis *redis.Client,
	validator *validator.Validator,
	authenticator auth.Authenticator,
) *AppContainer {
	// Repositories
	userRepository := repositories.NewUserRepository(db, cfg)
	otpRepository := repositories.NewOTPRepository(redis, cfg)
	sessionRepository := repositories.NewSessionRepository(db, cfg)
	refreshTokenRepository := repositories.NewRefreshTokenRepository(db, cfg)

	// Services
	authService := services.NewAuthService(
		userRepository,
		sessionRepository,
		refreshTokenRepository,
		otpRepository,
		logger,
		cfg,
		authenticator,
	)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, logger, validator)

	return &AppContainer{
		Config:        cfg,
		Logger:        logger,
		AuthHandler:   authHandler,
		Authenticator: authenticator,
	}
}

func (app *AppContainer) NewRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)               // Add a request ID to the context
	r.Use(middleware.RealIP)                  // Get the real IP address of the client
	r.Use(middleware.Recoverer)               // Recover from panics without crashing the server
	r.Use(middlewares.LogRequest(app.Logger)) // Log requests

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(app.Config.Server.ContextTimeout))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Redirect to swagger docs
		http.Redirect(w, r, "/docs", http.StatusMovedPermanently)
	})

	// TODO: Rate limit
	// TODO: Metrics

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		r.Route("/auth", func(r chi.Router) {
			r.Post("/otp/request", app.AuthHandler.RequestOtp)
			r.With(middlewares.RequestInfoMiddleware()).Post("/token", app.AuthHandler.Authenticate)
			r.Post("/token/refresh", app.AuthHandler.RefreshToken)
			r.Post("/logout", app.AuthHandler.Logout)
			r.Post("/logout/all", app.AuthHandler.LogoutAll)
			r.Post("/sessions", app.AuthHandler.GetSessions)
			r.Delete("/sessions/{id}", app.AuthHandler.DeleteSession)
		})
	})

	// TODO: CORS

	return r
}

func (app *AppContainer) StartServer(r *chi.Mux) error {
	server := &http.Server{
		Handler:      r,
		Addr:         ":" + app.Config.Port,
		WriteTimeout: app.Config.Server.WriteTimeout,
		ReadTimeout:  app.Config.Server.ReadTimeout,
		IdleTimeout:  app.Config.Server.IdleTimeout,
	}

	// Channel to listen for OS interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Channel to receive server errors
	serverErrors := make(chan error, 1)

	// Start server in a goroutine so we can listen for shutdown signals concurrently
	go func() {
		app.Logger.Info("Starting server", "port", app.Config.Port)
		serverErrors <- server.ListenAndServe()
	}()

	// Block until we receive an interrupt signal or server error
	select {
	case err := <-serverErrors:
		// Server failed to start or encountered an error
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}

	case sig := <-quit:
		// Received shutdown signal (Ctrl+C or kill command)
		app.Logger.Info("Shutting down server", "signal", sig)

		// Create context with 5-second timeout for graceful shutdown
		// This gives in-flight requests time to complete
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := server.Shutdown(ctx); err != nil {
			// If graceful shutdown fails, force close the server
			server.Close()
			return fmt.Errorf("server shutdown failed: %w", err)
		}

		app.Logger.Info("Server stopped gracefully")
	}

	return nil
}
