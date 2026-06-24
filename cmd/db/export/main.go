// Command export reads finished match results and all fair-play rows from prod
// (read-only; SELECT only) into a JSON snapshot file that `make db-seed-snapshot`
// replays into a dev DB. Uses PROD_DB_ADDRESS so DB_ADDRESS is never overridden.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"

	"github.com/fifawcp/api/cmd/db/snapshot"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/db"
	"github.com/fifawcp/api/internal/infrastructure/env"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	out := flag.String("out", "snapshot.json", "output file path")
	flag.Parse()

	cfg := config.NewConfig()
	logger := logging.NewSlogLogger(cfg)

	prodAddress := env.GetString("PROD_DB_ADDRESS", "")
	if prodAddress == "" {
		logger.Error("PROD_DB_ADDRESS is not set — add the prod connection string to .env before exporting")
		os.Exit(2)
	}

	logger.Warn("Connecting to PROD_DB_ADDRESS READ-ONLY (SELECT only) to export the real tournament snapshot")

	pgDB, err := db.NewPostgresDB(
		prodAddress,
		cfg.DB.MaxOpenConns,
		cfg.DB.MaxIdleConns,
		cfg.DB.MaxLifetime,
	)
	if err != nil {
		logger.Error("Error connecting to database", logging.Error, err.Error())
		os.Exit(1)
	}
	defer pgDB.Close()

	snap, err := snapshot.Export(context.Background(), pgDB)
	if err != nil {
		logger.Error("Error reading snapshot from database", logging.Error, err.Error())
		os.Exit(1)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		logger.Error("Error marshalling snapshot", logging.Error, err.Error())
		os.Exit(1)
	}

	if err := os.WriteFile(*out, data, 0o644); err != nil {
		logger.Error("Error writing snapshot file", logging.Error, err.Error())
		os.Exit(1)
	}

	logger.Info("Snapshot exported",
		"path", *out,
		"matches", len(snap.Matches),
		"fair_play", len(snap.FairPlay),
	)
}
