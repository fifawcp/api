package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/football"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/services"
)

const (
	syncInitialBuffer = 115 * time.Minute // fires after regular time + stoppage (~90 min playing + 15 min half time + 10 min buffer)
	syncRetryInterval = 15 * time.Minute  // covers extra time and penalty shootout window
	syncMaxRetries    = 5                 // 5 × 15 min = 75 min extra; total coverage ≈ 190 min from kickoff
)

type SyncMatchResultsJob struct {
	matchService   services.MatchServiceInterface
	footballClient *football.FootballClient
	fairPlayRepo   domain.MatchFairPlayRepository
	apiFixtureRepo domain.MatchAPIFixtureRepository
	logger         logging.Logger
}

func NewSyncMatchResultsJob(
	matchService services.MatchServiceInterface,
	footballClient *football.FootballClient,
	fairPlayRepo domain.MatchFairPlayRepository,
	apiFixtureRepo domain.MatchAPIFixtureRepository,
	logger logging.Logger,
) *SyncMatchResultsJob {
	return &SyncMatchResultsJob{
		matchService:   matchService,
		footballClient: footballClient,
		fairPlayRepo:   fairPlayRepo,
		apiFixtureRepo: apiFixtureRepo,
		logger:         logger,
	}
}

func (j *SyncMatchResultsJob) Name() string {
	return "sync:match_results"
}

func (j *SyncMatchResultsJob) Run(ctx context.Context) error {
	j.logger.Info("sync:match_results planning run started")

	// Get all scheduled matches
	matches, err := j.matchService.GetMatches(ctx, domain.MatchFilters{
		Status: domain.MatchStatusScheduled,
	})
	if err != nil {
		return fmt.Errorf("sync:match_results: fetch scheduled matches: %w", err)
	}

	scheduled := 0

	// For each scheduled match, resolve the api-football fixture ID and schedule a timer to sync the result
	for _, match := range matches {
		fixtureID, err := j.resolveFixtureID(ctx, match)
		if err != nil {
			j.logger.Warn("sync:match_results: cannot resolve fixture ID",
				"match_id", match.ID,
				"error", err,
			)
			continue
		}

		syncTime := match.KickoffAt.Add(syncInitialBuffer)
		delay := time.Until(syncTime)

		// Capture the match ID for the closure
		matchID := match.ID

		// Schedule the sync if the match is in the future
		if delay > 0 {
			time.AfterFunc(delay, func() {
				j.syncMatch(context.Background(), matchID, fixtureID, syncMaxRetries)
			})
		} else {
			// Sync the match immediately if it's in the past
			go j.syncMatch(ctx, matchID, fixtureID, syncMaxRetries)
		}

		scheduled++
	}

	j.logger.Info("sync:match_results planning run completed",
		"matches_scheduled", scheduled,
	)
	return nil
}

func (j *SyncMatchResultsJob) syncMatch(ctx context.Context, matchID, fixtureID int64, retriesLeft int) {
	fixture, err := j.footballClient.GetFixture(ctx, fixtureID)
	if err != nil {
		j.logger.Error("sync:match_results: get fixture failed",
			"fixture_id", fixtureID,
			"match_id", matchID,
			"error", err,
		)
		return
	}

	if !fixture.IsFinished() {
		// If the fixture is not finished, retry after the retry interval
		if retriesLeft > 0 {
			j.logger.Info("sync:match_results: match still in progress, will retry",
				"fixture_id", fixtureID,
				"match_id", matchID,
				"status", fixture.Fixture.Status.Short,
				"retries_left", retriesLeft,
			)

			time.AfterFunc(syncRetryInterval, func() {
				j.syncMatch(context.Background(), matchID, fixtureID, retriesLeft-1)
			})
		} else {
			// If the fixture is not finished and we've exhausted the retries, log a warning
			j.logger.Warn("sync:match_results: max retries exhausted, giving up",
				"fixture_id", fixtureID,
				"match_id", matchID,
				"status", fixture.Fixture.Status.Short,
			)
		}

		return
	}

	if err := j.persistResult(ctx, matchID, fixture); err != nil {
		j.logger.Error("sync:match_results: persist result failed",
			"fixture_id", fixtureID,
			"match_id", matchID,
			"error", err,
		)
		return
	}

	if err := j.persistFairPlay(ctx, matchID, fixture); err != nil {
		j.logger.Error("sync:match_results: persist fair play failed",
			"fixture_id", fixtureID,
			"match_id", matchID,
			"error", err,
		)
		return
	}

	j.logger.Info("sync:match_results: synced",
		"fixture_id", fixtureID,
		"match_id", matchID,
		"status", fixture.Fixture.Status.Short,
	)
}

func (j *SyncMatchResultsJob) persistResult(ctx context.Context, matchID int64, fixture *football.FixtureResponse) error {
	homeScore := 0
	if fixture.Goals.Home != nil {
		homeScore = *fixture.Goals.Home
	}
	awayScore := 0
	if fixture.Goals.Away != nil {
		awayScore = *fixture.Goals.Away
	}

	update := dtos.UpdateMatchResultDto{
		HomeScore: &homeScore,
		AwayScore: &awayScore,
	}

	// Attach penalty scores when the match was decided by a shootout.
	if fixture.Fixture.Status.Short == "PEN" {
		update.HomePenaltyScore = fixture.Score.Penalty.Home
		update.AwayPenaltyScore = fixture.Score.Penalty.Away
	}

	_, err := j.matchService.UpdateMatchResult(ctx, matchID, update)
	return err
}

func (j *SyncMatchResultsJob) persistFairPlay(ctx context.Context, matchID int64, fixture *football.FixtureResponse) error {
	cardSummaries := football.ParseCardEvents(fixture.Events)
	if len(cardSummaries) == 0 {
		return nil
	}

	var records []domain.MatchFairPlay
	for apiTeamID, summary := range cardSummaries {
		records = append(records, domain.MatchFairPlay{
			MatchID:                     matchID,
			TeamFIFACode:                football.APITeamIDToFIFACode[apiTeamID],
			YellowCards:                 summary.YellowCardCount,
			IndirectRedCards:            summary.IndirectRedCount,
			DirectRedCards:              summary.DirectRedCount,
			YellowCardAndDirectRedCards: summary.YellowCardAndDirectRedCardCount,
		})
	}

	// No update needed
	if len(records) == 0 {
		return nil
	}

	return j.fairPlayRepo.Upsert(ctx, records)
}

func (j *SyncMatchResultsJob) resolveFixtureID(ctx context.Context, match *domain.Match) (int64, error) {
	fixtureID, err := j.apiFixtureRepo.GetByMatchID(ctx, match.ID)
	if err == nil {
		return fixtureID, nil
	}
	if !errors.Is(err, domain.ErrMatchAPIFixtureNotFound) {
		return 0, fmt.Errorf("lookup fixture ID for match %d: %w", match.ID, err)
	}

	// Not in DB yet — only knockout matches can be in this state (group stage IDs are pre-seeded)
	// Discover the ID via the API and persist it for future planning runs
	if match.Teams.Home == nil || match.Teams.Away == nil {
		return 0, fmt.Errorf("match %d has unassigned teams", match.ID)
	}

	// Get the API team IDs from the FIFA codes
	homeAPITeamID := football.FifaCodeToAPITeamID[match.Teams.Home.FifaCode]
	awayAPITeamID := football.FifaCodeToAPITeamID[match.Teams.Away.FifaCode]

	// Discover the fixture ID via the API and persist it for future planning runs
	fixtureID, err = j.discoverAndPersistKnockoutFixtureID(ctx, match.ID, homeAPITeamID, awayAPITeamID)
	if err != nil {
		return 0, err
	}

	return fixtureID, nil
}

func (j *SyncMatchResultsJob) discoverAndPersistKnockoutFixtureID(ctx context.Context, matchID, homeAPITeamID, awayAPITeamID int64) (int64, error) {
	fixtures, err := j.footballClient.GetFixturesByTeam(ctx, homeAPITeamID)
	if err != nil {
		return 0, fmt.Errorf("get fixtures by team %d: %w", homeAPITeamID, err)
	}

	for _, fixture := range fixtures {
		// Check if the away team is the same as the away team in the match
		if fixture.Teams.Away.ID == awayAPITeamID {
			fixtureID := fixture.Fixture.ID
			// Persist the fixture ID for future planning runs
			if err := j.apiFixtureRepo.UpsertFixtureID(ctx, matchID, fixtureID); err != nil {
				j.logger.Warn("sync:match_results: failed to persist knockout fixture ID",
					"match_id", matchID,
					"fixture_id", fixtureID,
					"error", err,
				)
			}

			return fixtureID, nil
		}
	}

	return 0, fmt.Errorf("no knockout fixture found for home=%d away=%d", homeAPITeamID, awayAPITeamID)
}
