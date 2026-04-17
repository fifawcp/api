package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fifawcp/api/internal/handlers"
	"github.com/fifawcp/api/internal/infrastructure/auth"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/mailer"
	"github.com/fifawcp/api/internal/infrastructure/scheduler"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/jobs"
	"github.com/fifawcp/api/internal/repositories"
	"github.com/fifawcp/api/internal/services"
	"github.com/fifawcp/api/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type AppContainer struct {
	shutdownServer     func(*http.Server) error
	Config             *config.Config
	Logger             logging.Logger
	RateLimiters       *RateLimiters
	Scheduler          scheduler.Scheduler
	AuthHandler        *handlers.AuthHandler
	UserHandler        *handlers.UserHandler
	BoardHandler       *handlers.BoardHandler
	Authenticator      auth.Authenticator
	UserService        services.UserServiceInterface
	BoardMemberService services.BoardMemberServiceInterface
}

func NewAppContainer(
	cfg *config.Config,
	logger logging.Logger,
	db *sql.DB,
	redis *redis.Client,
	validator *validator.Validator,
	authenticator auth.Authenticator,
	mailer mailer.Mailer,
	scheduler scheduler.Scheduler,
) *AppContainer {
	// Repositories
	userRepository := repositories.NewUserRepository(db, cfg)
	sessionRepository := repositories.NewSessionRepository(db, cfg)
	refreshTokenRepository := repositories.NewRefreshTokenRepository(db, cfg)
	boardRepository := repositories.NewBoardRepository(db, cfg)
	boardMemberRepository := repositories.NewBoardMemberRepository(db, cfg)
	boardRankingRepository := repositories.NewBoardRankingRepository(db, cfg)

	// Storages
	otpStorage := storage.NewOTPStorage(redis, cfg)
	userStorage := storage.NewUserStorage(redis, cfg)

	// Services
	authService := services.NewAuthService(
		userRepository,
		sessionRepository,
		refreshTokenRepository,
		otpStorage,
		logger,
		cfg,
		authenticator,
		mailer,
	)
	userService := services.NewUserService(userRepository, userStorage, logger)
	boardService := services.NewBoardService(boardRepository)
	boardMemberService := services.NewBoardMemberService(boardRepository, boardMemberRepository)
	boardRankingService := services.NewBoardRankingService(boardRankingRepository)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, logger, validator, cfg)
	userHandler := handlers.NewUserHandler(userService, logger)
	boardHandler := handlers.NewBoardHandler(boardService, boardMemberService, boardRankingService, cfg, validator, logger)

	// Jobs
	cleanupSessionsJob := jobs.NewCleanupSessionsJob(sessionRepository, logger)

	// Schedule jobs
	if err := scheduler.RegisterJob(cfg.Cron.CleanupSessionsSchedule, cleanupSessionsJob); err != nil {
		logger.Error(
			"failed to register job",
			"job", "cleanup:expired-sessions",
			"error", err,
		)
	}

	rls := newRateLimiters(redis, &cfg.RateLimit)

	c := &AppContainer{
		Config:             cfg,
		Logger:             logger,
		Scheduler:          scheduler,
		RateLimiters:       rls,
		Authenticator:      authenticator,
		AuthHandler:        authHandler,
		UserHandler:        userHandler,
		BoardHandler:       boardHandler,
		UserService:        userService,
		BoardMemberService: boardMemberService,
	}
	c.shutdownServer = c.ShutdownServer
	return c
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

	// Jobs start ticking immediately alongside the server
	app.Scheduler.Start()

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
			// Stop the scheduler if the server fails to start
			app.Scheduler.Stop()
			return fmt.Errorf("server error: %w", err)
		}

	case sig := <-quit:
		// Received shutdown signal (Ctrl+C or kill command)
		app.Logger.Info("Shutting down server", "signal", sig)

		return app.shutdownServer(server)
	}

	return nil
}

func (app *AppContainer) ShutdownServer(server *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.Server.ShutdownTimeout)
	defer cancel()

	// Stop the scheduler before shutting down the server
	app.Scheduler.Stop()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		// If graceful shutdown fails, force close the server
		server.Close()
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	app.Logger.Info("Server stopped gracefully")
	return nil
}
