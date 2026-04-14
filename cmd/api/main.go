package main

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/joho/godotenv"
	"github.com/ncondes/fifawcp/docs"
	"github.com/ncondes/fifawcp/internal/app"
	"github.com/ncondes/fifawcp/internal/infrastructure/auth"
	"github.com/ncondes/fifawcp/internal/infrastructure/cache"
	"github.com/ncondes/fifawcp/internal/infrastructure/config"
	infradb "github.com/ncondes/fifawcp/internal/infrastructure/db"
	"github.com/ncondes/fifawcp/internal/infrastructure/logging"
	"github.com/ncondes/fifawcp/internal/infrastructure/mailer"
	"github.com/ncondes/fifawcp/internal/infrastructure/scheduler"
	"github.com/ncondes/fifawcp/internal/infrastructure/validator"
)

const version = "0.0.1"

//	@title			FIFA World Cup Pickems API
//	@version		0.0.0
//	@description	Passwordless authentication API with OTP, JWT, and multi-device session management.

//	@schemes	http https

//	@host
//	@BasePath	/api

//	@tag.name			auth
//	@tag.description	OTP request, token exchange, refresh, logout, and session management

//	@tag.name			users
//	@tag.description	Authenticated user profile and data

//	@tag.name			debug
//	@tag.description	Non-production only. These endpoints are not registered in production.

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Enter your JWT token in format: Bearer {token}
func main() {
	// Load environment variables from .env
	godotenv.Load()

	// Create config
	cfg := config.NewConfig()

	// Configure Swagger docs at runtime
	if u, err := url.Parse(cfg.APIBaseURL); err == nil {
		docs.SwaggerInfo.Host = u.Host
	}
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Version = version

	// Create Logger
	logger := logging.NewSlogLogger(cfg)

	// Connect to Database
	db, err := infradb.NewPostgresDB(
		cfg.DB.Address,
		cfg.DB.MaxOpenConns,
		cfg.DB.MaxIdleConns,
		cfg.DB.MaxLifetime,
	)
	if err != nil {
		logger.Error("Error connecting to database", "error", err)
		return
	}
	defer db.Close()
	logger.Info("Connected to database successfully")

	// Run database migrations
	if err := infradb.RunMigrations(db, "file://cmd/db/migrations"); err != nil {
		logger.Error("Error running database migrations", "error", err)
		return
	}
	logger.Info("Database migrations applied successfully")

	// Create Redis client
	redis, err := cache.NewRedisClient(cfg)
	if err != nil {
		logger.Error("Error connecting to Redis", "error", err)
		return
	}
	defer redis.Close()
	logger.Info("Connected to Redis successfully")

	// Create validator
	validator := validator.NewValidator()

	// Create JWT authenticator
	jwtAuthenticator := auth.NewJWTAuthenticator(
		cfg.JWT.Secret,
		cfg.JWT.Audience,
		cfg.JWT.Issuer,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	)

	// Create Mailer
	resendMailer := mailer.NewResendMailer(cfg)

	// Create scheduler
	scheduler := scheduler.NewCronScheduler(logger)

	// Create app container
	app := app.NewAppContainer(
		cfg,
		logger,
		db,
		redis,
		validator,
		jwtAuthenticator,
		resendMailer,
		scheduler,
	)

	// Create router
	router := app.NewRouter()

	// Start server
	if err := app.StartServer(router); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			logger.Info("Server stopped")
			return
		}

		logger.Error("Server error", "error", err)
	}
}
