package main

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/fifawcp/api/docs"
	"github.com/fifawcp/api/internal/app"
	"github.com/fifawcp/api/internal/infrastructure/auth"
	"github.com/fifawcp/api/internal/infrastructure/cache"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/db"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/mailer"
	"github.com/fifawcp/api/internal/infrastructure/oauth"
	"github.com/fifawcp/api/internal/infrastructure/scheduler"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/joho/godotenv"
)

const version = "0.0.1"

//	@title			FIFA World Cup Pickems API
//	@version		0.0.0
//	@description	Passwordless authentication API with OTP, JWT, and multi-device session management.

//	@schemes	http https

//	@host
//	@BasePath	/api

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Enter your JWT token in format: Bearer {token}
func main() {
	// Load environment variables from .env
	godotenv.Load()

	// Set timezone to GMT-5 (Eastern Tim)
	time.Local = time.FixedZone("GMT-5", -5*60*60)

	// Create config
	cfg := config.NewConfig()

	// Create Logger
	logger := logging.NewSlogLogger(cfg)

	// Create Google OIDC provider
	googleOIDCProvider, err := oauth.NewOIDCProvider(cfg.Auth.GoogleOAuth.Issuer)
	if err != nil {
		logger.Error("Error creating Google OIDC provider", "error", err)
		return
	}

	// Create Google OAuth client
	googleOAuthClient := oauth.NewGoogleOAuth2Client(googleOIDCProvider, cfg.Auth.GoogleOAuth)

	// Create Google identity verifier
	googleIDTokenVerifier := oauth.NewGoogleIDTokenVerifier(googleOIDCProvider, cfg.Auth.GoogleOAuth)

	// Configure Swagger docs at runtime
	if u, err := url.Parse(cfg.APIBaseURL); err == nil {
		docs.SwaggerInfo.Host = u.Host
	}
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Version = version

	// Connect to Database
	db, err := db.NewPostgresDB(
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
		googleOAuthClient,
		googleIDTokenVerifier,
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
