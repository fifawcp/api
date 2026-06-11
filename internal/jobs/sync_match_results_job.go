package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/football"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/services"
)

// SyncTiming controls the adaptive per-match poller.
type SyncTiming struct {
	FirstPollOffset time.Duration // delay from kickoff to the first poll
	NearEndInterval time.Duration // tight cadence near/at full time (floor of the clamp)
	MaxPollInterval time.Duration // sparsest cadence far from the end (ceiling of the clamp)
	ErrorBackoff    time.Duration // reschedule delay after a fetch error / non-200
	MaxPollWindow   time.Duration // hard stop: give up if now > kickoff + MaxPollWindow
}

// FixtureFetcher is the slice of the football client the job needs. Defined on
// the consumer side so the poller can be unit-tested with a mock.
type FixtureFetcher interface {
	GetFixture(ctx context.Context, fixtureID int64) (*football.FixtureResponse, error)
	GetFixturesByTeam(ctx context.Context, teamAPIID int64) ([]football.FixtureResponse, error)
}

type SyncMatchResultsJob struct {
	matchService   services.MatchServiceInterface
	fetcher        FixtureFetcher
	fairPlayRepo   domain.MatchFairPlayRepository
	apiFixtureRepo domain.MatchAPIFixtureRepository
	timing         SyncTiming
	logger         logging.Logger
	timersMutex    sync.Mutex
	timers         map[int64]*time.Timer
}

func NewSyncMatchResultsJob(
	matchService services.MatchServiceInterface,
	fetcher FixtureFetcher,
	fairPlayRepo domain.MatchFairPlayRepository,
	apiFixtureRepo domain.MatchAPIFixtureRepository,
	timing SyncTiming,
	logger logging.Logger,
) *SyncMatchResultsJob {
	return &SyncMatchResultsJob{
		matchService:   matchService,
		fetcher:        fetcher,
		fairPlayRepo:   fairPlayRepo,
		apiFixtureRepo: apiFixtureRepo,
		timing:         timing,
		logger:         logger,
		timers:         make(map[int64]*time.Timer),
	}
}

func (j *SyncMatchResultsJob) Name() string {
	return "sync:match_results"
}

func (j *SyncMatchResultsJob) Run(ctx context.Context) error {
	j.logger.Info("sync:match_results planning run started")

	matches, err := j.matchService.GetMatches(ctx, domain.MatchFilters{
		Status: domain.MatchStatusScheduled,
	})
	if err != nil {
		return fmt.Errorf("sync:match_results: fetch scheduled matches: %w", err)
	}

	scheduled := 0

	for _, match := range matches {
		fixtureID, err := j.resolveFixtureID(ctx, match)
		if err != nil {
			j.logger.Warn("sync:match_results: cannot resolve fixture ID",
				"match_id", match.ID,
				"error", err,
			)
			continue
		}

		// First poll fires at kickoff + FirstPollOffset; if that moment has
		// already passed, poll right away (startup backfill — an already-final
		// fixture persists on the first tick).
		delay := time.Until(match.KickoffAt.Add(j.timing.FirstPollOffset))
		if delay < 0 {
			delay = 0
		}

		j.schedulePoll(match.ID, fixtureID, match.KickoffAt, delay)
		scheduled++
	}

	j.logger.Info("sync:match_results planning run completed",
		"matches_scheduled", scheduled,
	)
	return nil
}

// schedulePoll registers (or replaces) the single pending poll timer for a match.
// The self-rescheduling poll re-arms through this same slot, so there is never
// more than one live timer per match and Stop() can cancel the whole chain.
func (j *SyncMatchResultsJob) schedulePoll(matchID, fixtureID int64, kickoffAt time.Time, delay time.Duration) {
	j.timersMutex.Lock()
	defer j.timersMutex.Unlock()

	if existing, ok := j.timers[matchID]; ok {
		existing.Stop()
	}

	j.timers[matchID] = time.AfterFunc(delay, func() {
		j.poll(context.Background(), matchID, fixtureID, kickoffAt)
	})
}

// clearTimer stops and forgets a match's timer once polling is done, so Stop()
// has nothing stale to cancel and the map does not grow unbounded.
func (j *SyncMatchResultsJob) clearTimer(matchID int64) {
	j.timersMutex.Lock()
	defer j.timersMutex.Unlock()

	if timer, ok := j.timers[matchID]; ok {
		timer.Stop()
		delete(j.timers, matchID)
	}
}

// Stop cancels all pending sync timers. Called on shutdown so timers do not fire
// during teardown; the next startup re-arms them from the scheduled matches.
func (j *SyncMatchResultsJob) Stop() {
	j.timersMutex.Lock()
	defer j.timersMutex.Unlock()

	for matchID, timer := range j.timers {
		timer.Stop()
		delete(j.timers, matchID)
	}
}

// poll fetches the fixture once and either persists the final result or re-arms
// itself at an adaptive cadence. Every non-terminal outcome — including a fetch
// error — reschedules through schedulePoll, so a transient failure never abandons
// the match and Stop() always has a single timer to cancel.
func (j *SyncMatchResultsJob) poll(ctx context.Context, matchID, fixtureID int64, kickoffAt time.Time) {
	if time.Now().After(kickoffAt.Add(j.timing.MaxPollWindow)) {
		j.clearTimer(matchID)
		j.logger.Warn("sync:match_results: poll window exceeded, giving up",
			"fixture_id", fixtureID,
			"match_id", matchID,
		)
		return
	}

	fixture, err := j.fetcher.GetFixture(ctx, fixtureID)
	if err != nil {
		// Reschedule rather than abandon: a transient error or 429 must not
		// permanently drop the match.
		j.logger.Warn("sync:match_results: get fixture failed, will retry",
			"fixture_id", fixtureID,
			"match_id", matchID,
			"error", err,
		)
		j.schedulePoll(matchID, fixtureID, kickoffAt, j.timing.ErrorBackoff)
		return
	}

	if fixture.IsFinished() {
		j.clearTimer(matchID)
		j.finalizeMatch(ctx, matchID, fixtureID, fixture)
		return
	}

	decision := nextPollDelay(
		fixture.Fixture.Status.Short, fixture.Fixture.Status.Elapsed,
		j.timing.NearEndInterval, j.timing.MaxPollInterval,
	)
	if decision.stop {
		j.clearTimer(matchID)
		j.logger.Warn("sync:match_results: terminal non-played status, giving up",
			"fixture_id", fixtureID,
			"match_id", matchID,
			"status", fixture.Fixture.Status.Short,
		)
		return
	}

	j.logger.Info("sync:match_results: in progress, will re-poll",
		"fixture_id", fixtureID,
		"match_id", matchID,
		"status", fixture.Fixture.Status.Short,
		"next_delay", decision.delay,
	)
	j.schedulePoll(matchID, fixtureID, kickoffAt, decision.delay)
}

// finalizeMatch persists a finished fixture's result and fair-play cards.
func (j *SyncMatchResultsJob) finalizeMatch(ctx context.Context, matchID, fixtureID int64, fixture *football.FixtureResponse) {
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

	// Not in DB yet — only knockout matches reach this (group-stage IDs are pre-seeded).
	if match.Teams.Home == nil || match.Teams.Away == nil {
		return 0, fmt.Errorf("match %d has unassigned teams", match.ID)
	}

	homeAPITeamID := football.FifaCodeToAPITeamID[match.Teams.Home.FifaCode]
	awayAPITeamID := football.FifaCodeToAPITeamID[match.Teams.Away.FifaCode]

	fixtureID, err = j.discoverAndPersistKnockoutFixtureID(ctx, match.ID, homeAPITeamID, awayAPITeamID)
	if err != nil {
		return 0, err
	}

	return fixtureID, nil
}

func (j *SyncMatchResultsJob) discoverAndPersistKnockoutFixtureID(ctx context.Context, matchID, homeAPITeamID, awayAPITeamID int64) (int64, error) {
	fixtures, err := j.fetcher.GetFixturesByTeam(ctx, homeAPITeamID)
	if err != nil {
		return 0, fmt.Errorf("get fixtures by team %d: %w", homeAPITeamID, err)
	}

	for _, fixture := range fixtures {
		if fixture.Teams.Away.ID == awayAPITeamID {
			fixtureID := fixture.Fixture.ID
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
