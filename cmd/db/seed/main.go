package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/db"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/repositories"
	"github.com/fifawcp/api/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	cfg := config.NewConfig()
	logger := logging.NewSlogLogger(cfg)

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

	groupStandingService := services.NewGroupStandingService(
		groupStandingRepository, matchRepository, matchFairPlayRepository, logger,
	)
	scoringService := services.NewScoringService(
		pickemRepository, matchScorePickRepository, scoreEventRepository,
		matchRepository, groupStandingRepository,
		cfg, logger,
	)
	competitionScoringService := services.NewCompetitionScoringService(
		competitionRepository, competitionScoreRepository, cfg, logger,
	)
	competitionService := services.NewCompetitionService(
		boardRepository, competitionRepository, competitionScoreRepository,
	)
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
		matchRepository, matchService, pickemService, competitionService,
	)

	flush := flag.Bool("flush", false, "")
	scenario := flag.String("scenario", "", "")
	flag.Parse()

	if *flush && *scenario != "" {
		logger.Error("-flush and -scenario are mutually exclusive")
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
	default:
		logger.Error("missing scenario: pass -scenario=<name> or use one of `make db-seed <scenario>` / `make db-flush`")
		os.Exit(2)
	}
}
