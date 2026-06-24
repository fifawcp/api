package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fifawcp/api/cmd/db/snapshot"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/db"
	"github.com/fifawcp/api/internal/infrastructure/env"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/repositories"
	"github.com/fifawcp/api/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	cfg := config.NewConfig()
	logger := logging.NewSlogLogger(cfg)

	// The seeder is destructive (Flush deletes users/boards, resets matches) and
	// ships to prod as /seed. Refuse to run against production — both by the
	// environment it runs in and by the target DB matching the known prod one.
	if cfg.IsProd() {
		logger.Error("refusing to seed: ENV is production (the seeder is destructive and dev-only)")
		os.Exit(2)
	}
	if prodAddress := env.GetString("PROD_DB_ADDRESS", ""); prodAddress != "" && cfg.DB.Address == prodAddress {
		logger.Error("refusing to seed: DB_ADDRESS matches PROD_DB_ADDRESS (the seeder is destructive and dev-only)")
		os.Exit(2)
	}

	pgDB, err := db.NewPostgresDB(
		cfg.DB.Address,
		cfg.DB.MaxOpenConns,
		cfg.DB.MaxIdleConns,
		cfg.DB.MaxLifetime,
	)
	if err != nil {
		logger.Error("Error connecting to database", logging.Error, err.Error())
		os.Exit(1)
	}
	defer pgDB.Close()
	logger.Info("Connected to database successfully")

	// Load teams once so the match/standings repos share a populated TeamLookup.
	teamRepository := repositories.NewTeamRepository(pgDB, cfg)
	teams, err := teamRepository.GetAllTeams(context.Background())
	if err != nil {
		logger.Error("Error loading teams", logging.Error, err.Error())
		os.Exit(1)
	}
	teamLookup := domain.NewTeamLookup(teams)

	userRepository := repositories.NewUserRepository(pgDB, cfg)
	boardRepository := repositories.NewBoardRepository(pgDB, cfg)
	boardMemberRepository := repositories.NewBoardMemberRepository(pgDB, cfg)
	pickemRepository := repositories.NewPickemRepository(pgDB, cfg)
	matchRepository := repositories.NewMatchRepository(pgDB, cfg, teamLookup)
	groupStandingRepository := repositories.NewGroupStandingRepository(pgDB, cfg, teamLookup)
	matchFairPlayRepository := repositories.NewMatchFairPlayRepository(pgDB, cfg)
	matchScorePickRepository := repositories.NewMatchScorePickRepository(pgDB, cfg)
	scoreEventRepository := repositories.NewScoreEventRepository(pgDB, cfg)
	competitionRepository := repositories.NewCompetitionRepository(pgDB, cfg)
	competitionScoreRepository := repositories.NewCompetitionScoreRepository(pgDB, cfg)
	awardPickRepository := repositories.NewAwardPickRepository(pgDB, cfg, teamLookup)

	groupStandingService := services.NewGroupStandingService(
		groupStandingRepository, matchRepository, matchFairPlayRepository, logger,
	)
	scoringService := services.NewScoringService(
		pickemRepository, matchScorePickRepository, scoreEventRepository,
		awardPickRepository,
		matchRepository, groupStandingRepository,
		cfg, logger,
	)
	competitionScoringService := services.NewCompetitionScoringService(
		competitionScoreRepository, cfg, logger,
	)
	competitionService := services.NewCompetitionService(
		boardRepository, competitionRepository, competitionScoreRepository,
	)
	boardService := services.NewBoardService(boardRepository, competitionRepository)
	matchService := services.NewMatchService(
		matchRepository, groupStandingRepository,
		groupStandingService, scoringService, competitionScoringService, logger,
	)
	pickemService := services.NewPickemService(
		pickemRepository, teams, time.Now().Add(365*24*time.Hour), cfg, logger,
	)

	seeder := NewSeeder(
		pgDB, logger,
		userRepository, boardRepository, boardMemberRepository, pickemRepository,
		matchRepository, awardPickRepository, matchFairPlayRepository,
		matchService, pickemService, boardService, competitionService,
	)

	flush := flag.Bool("flush", false, "")
	scenario := flag.String("scenario", "", "")
	snapshotPath := flag.String("snapshot", "", "")
	fromProd := flag.Bool("from-prod", false, "")
	flag.Parse()

	selectedModes := 0
	if *flush {
		selectedModes++
	}
	if *scenario != "" {
		selectedModes++
	}
	if *snapshotPath != "" {
		selectedModes++
	}
	if *fromProd {
		selectedModes++
	}
	if selectedModes > 1 {
		logger.Error("-flush, -scenario, -snapshot and -from-prod are mutually exclusive")
		os.Exit(2)
	}

	switch {
	case *flush:
		seeder.Flush()
	case *scenario != "":
		if err := seeder.RunScenario(context.Background(), *scenario); err != nil {
			logger.Error("Scenario seed failed", logging.Error, err.Error())
			os.Exit(1)
		}
	case *snapshotPath != "":
		if err := seeder.RunSnapshot(context.Background(), *snapshotPath); err != nil {
			logger.Error("Snapshot seed failed", logging.Error, err.Error())
			os.Exit(1)
		}
	case *fromProd:
		if err := seedFromProd(context.Background(), cfg, logger, seeder); err != nil {
			logger.Error("Seed from prod failed", logging.Error, err.Error())
			os.Exit(1)
		}
	default:
		logger.Error("missing mode: pass -scenario=<name>, -snapshot=<path>, -from-prod, or -flush")
		os.Exit(2)
	}
}

// seedFromProd reads the real snapshot live from PROD_DB_ADDRESS (read-only) and
// replays it into the dev DB. The guards above already refuse a prod target.
func seedFromProd(ctx context.Context, cfg *config.Config, logger logging.Logger, seeder *Seeder) error {
	prodAddress := env.GetString("PROD_DB_ADDRESS", "")
	if prodAddress == "" {
		return errors.New("PROD_DB_ADDRESS is not set — add the prod connection string before seeding from prod")
	}

	logger.Warn("Connecting to PROD_DB_ADDRESS READ-ONLY (SELECT only) to source the real tournament snapshot")

	prodDB, err := db.NewPostgresDB(
		prodAddress,
		cfg.DB.MaxOpenConns,
		cfg.DB.MaxIdleConns,
		cfg.DB.MaxLifetime,
	)
	if err != nil {
		return fmt.Errorf("connect to prod: %w", err)
	}
	defer prodDB.Close()

	snap, err := snapshot.Export(ctx, prodDB)
	if err != nil {
		return fmt.Errorf("read prod snapshot: %w", err)
	}

	return seeder.RunSnapshotData(ctx, snap)
}
