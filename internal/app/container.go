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
	"github.com/go-chi/cors"
	"github.com/ncondes/fifawcp/internal/handlers"
	"github.com/ncondes/fifawcp/internal/infrastructure/auth"
	"github.com/ncondes/fifawcp/internal/infrastructure/config"
	"github.com/ncondes/fifawcp/internal/infrastructure/logging"
	"github.com/ncondes/fifawcp/internal/infrastructure/mailer"
	"github.com/ncondes/fifawcp/internal/infrastructure/middlewares"
	"github.com/ncondes/fifawcp/internal/infrastructure/scheduler"
	"github.com/ncondes/fifawcp/internal/infrastructure/validator"
	"github.com/ncondes/fifawcp/internal/jobs"
	"github.com/ncondes/fifawcp/internal/repositories"
	"github.com/ncondes/fifawcp/internal/services"
	"github.com/ncondes/fifawcp/internal/storage"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
)

type AppContainer struct {
	Config        *config.Config
	Logger        logging.Logger
	AuthHandler   *handlers.AuthHandler
	UserHandler   *handlers.UserHandler
	UserService   services.UserServiceInterface
	Authenticator auth.Authenticator
	Scheduler     scheduler.Scheduler
	RateLimiters  *RateLimiters
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

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, logger, validator, cfg)
	userHandler := handlers.NewUserHandler(userService, logger)

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

	return &AppContainer{
		Config:        cfg,
		Logger:        logger,
		AuthHandler:   authHandler,
		UserHandler:   userHandler,
		UserService:   userService,
		Authenticator: authenticator,
		Scheduler:     scheduler,
		RateLimiters:  rls,
	}
}

func (app *AppContainer) NewRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)                         // Add a request ID to the context
	r.Use(middleware.RealIP)                            // Get the real IP address of the client
	r.Use(middleware.Recoverer)                         // Recover from panics without crashing the server
	r.Use(middlewares.LogRequestMiddleware(app.Logger)) // Log requests

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(app.Config.Server.ContextTimeout))

	// Security headers
	r.Use(middlewares.SecurityHeadersMiddleware(app.Config))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   app.Config.Server.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// TODO: Metrics

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(app.Config.APIBaseURL+"/swagger/doc.json"),
	))

	r.Route("/debug", func(r chi.Router) {
		if !app.Config.IsProd() {
			debugHandler := handlers.NewDebugHandler(app.Config)
			r.Get("/auth/otp/request/{identifier}", debugHandler.RequestOtp)
		}
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		r.Route("/auth", func(r chi.Router) {
			r.With(middlewares.RateLimitByIPMiddleware(
				app.RateLimiters.StrictIP,
				"auth:otp:request",
				app.Logger,
			)).Post("/otp/request", app.AuthHandler.RequestOtp)

			r.With(
				middlewares.RateLimitByIPMiddleware(
					app.RateLimiters.ModerateIP,
					"auth:token",
					app.Logger,
				),
				middlewares.RequestInfoMiddleware(),
			).Post("/token", app.AuthHandler.Authenticate)

			r.With(middlewares.RateLimitByIPMiddleware(
				app.RateLimiters.RelaxedIP,
				"auth:token:refresh",
				app.Logger,
			)).Post("/token/refresh", app.AuthHandler.RefreshToken)

			r.Post("/logout", app.AuthHandler.Logout)
			r.Post("/logout/all", app.AuthHandler.LogoutAll)
			r.Get("/sessions", app.AuthHandler.GetSessions)
			r.With(middlewares.AuthMiddleware(
				app.Authenticator,
				app.UserService,
				app.Logger,
			)).Delete("/sessions/{id}", app.AuthHandler.DeleteSession)
		})

		r.Route("/users", func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware(app.Authenticator, app.UserService, app.Logger))
			r.Get("/profile", app.UserHandler.GetProfile)
		})
	})

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

		// Create context with 5-second timeout for graceful shutdown
		// This gives in-flight requests time to complete
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	}

	return nil
}
