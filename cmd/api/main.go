package main

import (
	"errors"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/ncondes/fifa-world-cup-pickems/internal/app"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/auth"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/cache"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/config"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/db"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/logging"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/validator"
)

func main() {
	// Load environment variables from .env
	godotenv.Load()

	// Create config
	cfg := config.NewConfig()

	// Create Logger
	logger := logging.NewSlogLogger(cfg)

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

	// Create app container
	app := app.NewAppContainer(
		cfg,
		logger,
		db,
		redis,
		validator,
		jwtAuthenticator,
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
