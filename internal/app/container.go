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

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/handlers"
	"github.com/fifawcp/api/internal/infrastructure/auth"
	"github.com/fifawcp/api/internal/infrastructure/cache"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/db"
	"github.com/fifawcp/api/internal/infrastructure/football"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/mailer"
	"github.com/fifawcp/api/internal/infrastructure/oauth"
	"github.com/fifawcp/api/internal/infrastructure/ratelimit"
	"github.com/fifawcp/api/internal/infrastructure/scheduler"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/jobs"
	"github.com/fifawcp/api/internal/repositories"
	"github.com/fifawcp/api/internal/services"
	"github.com/fifawcp/api/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type Container struct {
	shutdownServer func(*http.Server) error

	// infrastructure
	db          *sql.DB
	redis       *redis.Client
	Config      *config.Config
	Logger      logging.Logger
	AuditLogger logging.AuditLogger

	// core deps
	validator            *validator.Validator
	mailer               mailer.Mailer
	Authenticator        auth.Authenticator
	Scheduler            scheduler.Scheduler
	GoogleOauthConfig    domain.OAuth2Client
	OIDCIdentityVerifier domain.IDTokenVerifier

	// repositories
	userRepository             domain.UserRepository
	sessionRepository          domain.SessionRepository
	refreshTokenRepository     domain.RefreshTokenRepository
	boardRepository            domain.BoardRepository
	boardMemberRepository      domain.BoardMemberRepository
	groupStandingRepository    domain.GroupStandingRepository
	matchRepository            domain.MatchRepository
	matchFairPlayRepository    domain.MatchFairPlayRepository
	matchAPIFixtureRepository  domain.MatchAPIFixtureRepository
	oauthAccountRepository     domain.OAuthAccountRepository
	teamRepository             domain.TeamRepository
	pickemRepository           domain.PickemRepository
	matchScorePickRepository   domain.MatchScorePickRepository
	scoreEventRepository       domain.ScoreEventRepository
	competitionRepository      domain.CompetitionRepository
	competitionScoreRepository domain.CompetitionScoreRepository
	playerRepository           domain.PlayerRepository
	awardPickRepository        domain.AwardPickRepository

	// startup data loaded once at startup; consumed by services
	teams                   []*domain.Team
	teamLookup              *domain.TeamLookup
	firstKickoff            time.Time
	globalPickemCompetition *domain.Competition
	globalMatchCompetition  *domain.Competition

	// storages
	otpStorage        *storage.OTPStorage
	userStorage       *storage.UserStorage
	oauthStateStorage *storage.OAuthStorage

	// services
	authService               services.AuthServiceInterface
	boardService              services.BoardServiceInterface
	groupStandingService      services.GroupStandingServiceInterface
	matchService              services.MatchServiceInterface
	matchScorePickService     services.MatchScorePickServiceInterface
	oauthService              services.OAuthServiceInterface
	UserService               services.UserServiceInterface
	BoardMemberService        services.BoardMemberServiceInterface
	pickemService             services.PickemServiceInterface
	pickemScoringService      services.ScoringServiceInterface
	competitionService        services.CompetitionServiceInterface
	competitionScoringService services.CompetitionScoringServiceInterface
	dashboardService          services.DashboardServiceInterface
	playerService             services.PlayerServiceInterface
	awardService              services.AwardServiceInterface

	// handlers
	RateLimiters       *RateLimiters
	AuthHandler        *handlers.AuthHandler
	OAuthHandler       *handlers.OAuthHandler
	UserHandler        *handlers.UserHandler
	BoardHandler       *handlers.BoardHandler
	GroupHandler       *handlers.GroupStandingHandler
	MatchHandler       *handlers.MatchHandler
	AdminHandler       *handlers.AdminHandler
	PickemHandler      *handlers.PickemHandler
	CompetitionHandler *handlers.CompetitionHandler
	DashboardHandler   *handlers.DashboardHandler
	PlayerHandler      *handlers.PlayerHandler
	AwardHandler       *handlers.AwardHandler
}

func NewContainer(cfg *config.Config) (*Container, error) {
	c := &Container{Config: cfg}

	if err := c.initInfrastructure(cfg); err != nil {
		return nil, fmt.Errorf("infrastructure: %w", err)
	}
	if err := c.initCoreDeps(cfg); err != nil {
		return nil, fmt.Errorf("core dependencies: %w", err)
	}
	c.initRepositories()
	c.initStorages()
	if err := c.initStartupData(); err != nil {
		return nil, fmt.Errorf("startup data: %w", err)
	}
	c.initServices()
	c.initHandlers()
	c.initJobs()

	c.RateLimiters = newRateLimiters(c.redis, &cfg.RateLimit)
	c.shutdownServer = c.ShutdownServer

	return c, nil
}

func (c *Container) initInfrastructure(cfg *config.Config) error {
	c.Logger = logging.NewSlogLogger(cfg)
	c.AuditLogger = logging.NewAuditLogger(c.Logger)

	pgDB, err := db.NewPostgresDB(
		cfg.DB.Address,
		cfg.DB.MaxOpenConns,
		cfg.DB.MaxIdleConns,
		cfg.DB.MaxLifetime,
	)
	if err != nil {
		c.Logger.Error(
			"Error connecting to database",
			logging.Error, err.Error(),
		)
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	c.db = pgDB
	c.Logger.Info("Connected to database successfully")

	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		c.Logger.Error(
			"Error connecting to Redis",
			logging.Error, err.Error(),
		)
		return fmt.Errorf("connecting to Redis: %w", err)
	}
	c.redis = redisClient
	c.Logger.Info("Connected to Redis successfully")

	return nil
}

func (c *Container) initCoreDeps(cfg *config.Config) error {
	c.validator = validator.NewValidator()
	c.Authenticator = auth.NewJWTAuthenticator(
		cfg.JWT.Secret,
		cfg.JWT.Audience,
		cfg.JWT.Issuer,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	)
	c.mailer = mailer.NewResendMailer(cfg)
	c.Scheduler = scheduler.NewCronScheduler(c.Logger)

	googleOIDCProvider, err := oauth.NewOIDCProvider(cfg.Auth.GoogleOAuth.Issuer)
	if err != nil {
		c.Logger.Error(
			"Error creating Google OIDC provider",
			logging.Error, err.Error(),
		)
		return fmt.Errorf("creating OIDC provider: %w", err)
	}
	c.GoogleOauthConfig = oauth.NewGoogleOAuth2Client(googleOIDCProvider, cfg.Auth.GoogleOAuth)
	c.OIDCIdentityVerifier = oauth.NewGoogleIDTokenVerifier(googleOIDCProvider, cfg.Auth.GoogleOAuth)

	return nil
}

func (c *Container) initRepositories() {
	c.userRepository = repositories.NewUserRepository(c.db, c.Config)
	c.sessionRepository = repositories.NewSessionRepository(c.db, c.Config)
	c.refreshTokenRepository = repositories.NewRefreshTokenRepository(c.db, c.Config)
	c.boardRepository = repositories.NewBoardRepository(c.db, c.Config)
	c.boardMemberRepository = repositories.NewBoardMemberRepository(c.db, c.Config)
	c.oauthAccountRepository = repositories.NewOAuthAccountRepository(c.db, c.Config)
	c.teamRepository = repositories.NewTeamRepository(c.db, c.Config)
	c.pickemRepository = repositories.NewPickemRepository(c.db, c.Config)
	c.matchScorePickRepository = repositories.NewMatchScorePickRepository(c.db, c.Config)
	c.scoreEventRepository = repositories.NewScoreEventRepository(c.db, c.Config)

	c.teamLookup = domain.NewTeamLookup(nil)
	c.matchRepository = repositories.NewMatchRepository(c.db, c.Config, c.teamLookup)
	c.groupStandingRepository = repositories.NewGroupStandingRepository(c.db, c.Config, c.teamLookup)
	c.matchFairPlayRepository = repositories.NewMatchFairPlayRepository(c.db, c.Config)
	c.matchAPIFixtureRepository = repositories.NewMatchAPIFixtureRepository(c.db, c.Config)
	c.competitionRepository = repositories.NewCompetitionRepository(c.db, c.Config)
	c.competitionScoreRepository = repositories.NewCompetitionScoreRepository(c.db, c.Config)
	c.playerRepository = repositories.NewPlayerRepository(c.db, c.Config, c.teamLookup)
	c.awardPickRepository = repositories.NewAwardPickRepository(c.db, c.Config, c.teamLookup)
}

func (c *Container) initStartupData() error {
	teams, err := c.teamRepository.GetAllTeams(context.Background())
	if err != nil {
		return fmt.Errorf("loading team catalog: %w", err)
	}
	c.teams = teams
	c.teamLookup.Set(teams)

	firstKickoff, err := c.matchRepository.GetFirstGroupStageMatchKickoff(context.Background())
	if err != nil {
		return fmt.Errorf("loading first group stage match kickoff: %w", err)
	}
	c.firstKickoff = firstKickoff

	globalPickemCompetition, globalMatchCompetition, err := c.competitionRepository.GetGlobalCompetitions(context.Background())
	if err != nil {
		return fmt.Errorf("loading global competitions: %w", err)
	}
	c.globalPickemCompetition = globalPickemCompetition
	c.globalMatchCompetition = globalMatchCompetition

	return nil
}

func (c *Container) initStorages() {
	c.otpStorage = storage.NewOTPStorage(c.redis, c.Config)
	c.userStorage = storage.NewUserStorage(c.redis, c.Config)
	c.oauthStateStorage = storage.NewOAuthStorage(c.redis, c.Config)
}

func (c *Container) initServices() {
	c.authService = services.NewAuthService(
		c.userRepository, c.sessionRepository, c.refreshTokenRepository,
		c.otpStorage, c.Logger, c.Config, c.Authenticator, c.mailer,
	)
	c.UserService = services.NewUserService(c.userRepository, c.userStorage, c.Logger)
	c.boardService = services.NewBoardService(c.boardRepository, c.competitionRepository)
	c.BoardMemberService = services.NewBoardMemberService(
		c.boardRepository, c.boardMemberRepository,
	)
	c.groupStandingService = services.NewGroupStandingService(
		c.groupStandingRepository, c.matchRepository, c.matchFairPlayRepository, c.Logger,
	)
	c.pickemScoringService = services.NewScoringService(
		c.pickemRepository, c.matchScorePickRepository, c.scoreEventRepository,
		c.awardPickRepository,
		c.matchRepository, c.groupStandingRepository,
		c.Config, c.Logger,
	)
	c.pickemService = services.NewPickemService(
		c.pickemRepository, c.teams, c.firstKickoff, c.Config, c.Logger,
	)
	c.matchScorePickService = services.NewMatchScorePickService(
		c.matchScorePickRepository, c.matchRepository,
	)

	c.competitionScoringService = services.NewCompetitionScoringService(
		c.competitionRepository, c.competitionScoreRepository, c.Config, c.Logger,
	)
	c.competitionService = services.NewCompetitionService(
		c.boardRepository, c.competitionRepository, c.competitionScoreRepository,
	)

	c.matchService = services.NewMatchService(
		c.matchRepository, c.groupStandingRepository,
		c.groupStandingService, c.pickemScoringService, c.competitionScoringService, c.Logger,
	)
	c.oauthService = services.NewOAuthService(
		c.oauthStateStorage, c.GoogleOauthConfig, c.OIDCIdentityVerifier,
		c.oauthAccountRepository, c.userRepository, c.authService,
	)

	c.playerService = services.NewPlayerService(c.playerRepository)
	c.awardService = services.NewAwardService(
		c.awardPickRepository, c.playerRepository,
		c.pickemScoringService, c.competitionScoringService,
		c.firstKickoff, c.Config, c.Logger,
	)

	c.dashboardService = services.NewDashboardService(
		c.pickemService,
		c.awardService,
		c.matchScorePickRepository,
		c.matchRepository,
		c.competitionScoreRepository,
		c.globalPickemCompetition,
		c.globalMatchCompetition,
	)
}

func (c *Container) initHandlers() {
	c.AuthHandler = handlers.NewAuthHandler(c.authService, c.Logger, c.validator, c.Config)
	c.UserHandler = handlers.NewUserHandler(c.UserService, c.Logger, c.validator)
	c.BoardHandler = handlers.NewBoardHandler(c.boardService, c.BoardMemberService, c.Config, c.validator, c.Logger)
	c.GroupHandler = handlers.NewGroupStandingHandler(c.groupStandingService, c.Logger)
	c.MatchHandler = handlers.NewMatchHandler(c.matchService, c.matchScorePickService, c.Logger, c.validator)
	c.AdminHandler = handlers.NewAdminHandler(c.matchService, c.groupStandingService, c.pickemScoringService, c.awardService, c.Logger, c.AuditLogger, c.validator)
	c.OAuthHandler = handlers.NewOAuthHandler(c.oauthService, c.Logger, c.Config)
	c.PickemHandler = handlers.NewPickemHandler(c.pickemService, c.Logger, c.validator)
	c.CompetitionHandler = handlers.NewCompetitionHandler(c.competitionService, c.Config, c.validator, c.Logger)
	c.DashboardHandler = handlers.NewDashboardHandler(c.dashboardService, c.Logger)
	c.PlayerHandler = handlers.NewPlayerHandler(c.playerService, c.Logger)
	c.AwardHandler = handlers.NewAwardHandler(c.awardService, c.Logger, c.validator)
}

func (c *Container) initJobs() {
	if err := c.Scheduler.RegisterJob(c.Config.Cron.CleanupSessionsSchedule, jobs.NewCleanupSessionsJob(c.sessionRepository, c.refreshTokenRepository, c.Config.JWT.RefreshGraceWindow, c.Logger)); err != nil {
		c.Logger.Error(
			"failed to register job",
			"job", "cleanup:expired-sessions",
			logging.Error, err.Error(),
		)
	}

	footballClient := football.NewFootballClient(c.Config.FootballAPI)
	syncMatchResultsJob := jobs.NewSyncMatchResultsJob(
		c.matchService,
		footballClient,
		c.matchFairPlayRepository,
		c.matchAPIFixtureRepository,
		c.Logger,
	)

	// Run once at startup to register timers for upcoming matches and backfill any
	// matches whose sync window has already passed (e.g. after a server restart)
	go syncMatchResultsJob.Run(context.Background())

	// Re-run at midnight to pick up newly-assigned knockout teams and re-register timers
	if err := c.Scheduler.RegisterJob(c.Config.Cron.SyncMatchResultsSchedule, syncMatchResultsJob); err != nil {
		c.Logger.Error(
			"failed to register job",
			"job", "sync:match_results",
			logging.Error, err.Error(),
		)
	}
}

func (c *Container) Cleanup() {
	if c.db != nil {
		c.db.Close()
	}

	if c.redis != nil {
		c.redis.Close()
	}
}

func (c *Container) StartServer(r *chi.Mux) error {
	server := &http.Server{
		Handler:      r,
		Addr:         ":" + c.Config.Port,
		WriteTimeout: c.Config.Server.WriteTimeout,
		ReadTimeout:  c.Config.Server.ReadTimeout,
		IdleTimeout:  c.Config.Server.IdleTimeout,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)

	c.Scheduler.Start()

	go func() {
		c.Logger.Info("Starting server", "port", c.Config.Port)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		c.Scheduler.Stop()
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		c.Logger.Info("Shutting down server", "signal", sig)
		return c.shutdownServer(server)
	}
}

func (c *Container) ShutdownServer(server *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.Config.Server.ShutdownTimeout)
	defer cancel()

	c.Scheduler.Stop()

	if err := server.Shutdown(ctx); err != nil {
		server.Close()
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	c.Logger.Info("Server stopped gracefully")
	return nil
}

type RateLimiters struct {
	StrictIP   ratelimit.RateLimiter
	ModerateIP ratelimit.RateLimiter
	RelaxedIP  ratelimit.RateLimiter
}

func newRateLimiters(rc *redis.Client, cfg *config.RateLimitConfig) *RateLimiters {
	if !cfg.Enabled || rc == nil {
		return &RateLimiters{}
	}

	return &RateLimiters{
		StrictIP:   ratelimit.NewRedisRateLimiter(rc, cfg.StrictIP),
		ModerateIP: ratelimit.NewRedisRateLimiter(rc, cfg.ModerateIP),
		RelaxedIP:  ratelimit.NewRedisRateLimiter(rc, cfg.RelaxedIP),
	}
}
