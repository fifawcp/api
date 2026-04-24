package main

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/fifawcp/api/docs"
	"github.com/fifawcp/api/internal/app"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/joho/godotenv"
)

const version = "0.0.1"

// @title			FIFA World Cup Pickems API
// @version		0.0.0
// @description	Passwordless authentication API with OTP, JWT, and multi-device session management.

// @schemes	http https

// @host
// @BasePath	/api

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Enter your JWT token in format: Bearer {token}
func main() {
	godotenv.Load()

	// Set timezone to GMT-5 (Eastern Time)
	time.Local = time.FixedZone("GMT-5", -5*60*60)

	cfg := config.NewConfig()

	if u, err := url.Parse(cfg.APIBaseURL); err == nil {
		docs.SwaggerInfo.Host = u.Host
	}
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Version = version

	container, err := app.NewContainer(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer container.Cleanup()

	router := container.NewRouter()

	if err := container.StartServer(router); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		log.Fatal(err)
	}
}
