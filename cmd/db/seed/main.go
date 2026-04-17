package main

import (
	"flag"
	"os"

	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/db"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/repositories"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	cfg := config.NewConfig()

	logger := logging.NewSlogLogger(cfg)

	db, err := db.NewPostgresDB(
		cfg.DB.Address,
		cfg.DB.MaxOpenConns,
		cfg.DB.MaxIdleConns,
		cfg.DB.MaxLifetime,
	)
	if err != nil {
		logger.Error("Error connecting to database", "error", err)
		os.Exit(1)
	}

	defer db.Close()
	logger.Info("Connected to database successfully")

	userRepository := repositories.NewUserRepository(db, cfg)
	seeder := NewSeeder(db, logger, userRepository)

	flush := flag.Bool("flush", false, "Flush the database without seeding")
	flag.Parse()

	if *flush {
		seeder.Flush()
	} else {
		seeder.Run()
	}
}
