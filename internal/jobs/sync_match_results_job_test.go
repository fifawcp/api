package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/football"
	"github.com/fifawcp/api/internal/test/mocks"
)

// neverEndingTiming keeps every rescheduled timer far in the future so it cannot
// fire during a test; assertions look at j.timers membership, not side effects.
func neverEndingTiming() SyncTiming {
	return SyncTiming{
		FirstPollOffset: 75 * time.Minute,
		NearEndInterval: time.Hour,
		MaxPollInterval: time.Hour,
		ErrorBackoff:    time.Hour,
		MaxPollWindow:   4 * time.Hour,
	}
}

// TestSchedulePoll_ReplacesExistingTimer verifies that re-scheduling the same
// match stops the previous timer and keeps exactly one pending timer, so daily
// re-runs and self-rescheduling polls do not accumulate duplicate timers.
func TestSchedulePoll_ReplacesExistingTimer(t *testing.T) {
	job := &SyncMatchResultsJob{timers: make(map[int64]*time.Timer)}

	const matchID int64 = 1
	const fixtureID int64 = 10
	kickoff := time.Now()

	// A long delay guarantees neither timer fires during the test.
	job.schedulePoll(matchID, fixtureID, kickoff, time.Hour)
	firstTimer := job.timers[matchID]

	job.schedulePoll(matchID, fixtureID, kickoff, time.Hour)
	secondTimer := job.timers[matchID]

	if len(job.timers) != 1 {
		t.Fatalf("expected exactly 1 pending timer, got %d", len(job.timers))
	}
	if firstTimer == secondTimer {
		t.Fatal("expected the timer to be replaced on re-schedule")
	}
	if firstTimer.Stop() {
		t.Fatal("expected the previous timer to have been stopped")
	}
}

// TestStop_CancelsAllPendingTimers verifies shutdown cancels every pending timer.
func TestStop_CancelsAllPendingTimers(t *testing.T) {
	job := &SyncMatchResultsJob{timers: make(map[int64]*time.Timer)}
	now := time.Now()

	job.schedulePoll(1, 10, now, time.Hour)
	job.schedulePoll(2, 20, now, time.Hour)
	timer := job.timers[1]

	job.Stop()

	if len(job.timers) != 0 {
		t.Fatalf("expected all timers cleared, got %d", len(job.timers))
	}
	if timer.Stop() {
		t.Fatal("expected pending timer to have been stopped")
	}
}

// TestPoll_PersistsAndClearsTimer_WhenFinal verifies the happy path: a finished
// fixture is finalized via the sync service and its timer is cleared.
func TestPoll_PersistsAndClearsTimer_WhenFinal(t *testing.T) {
	fetcher := &mocks.MockFixtureFetcher{
		GetFixtureFunc: func(_ context.Context, fixtureID int64) (*football.FixtureResponse, error) {
			return &football.FixtureResponse{
				Fixture: football.FixtureInfo{ID: fixtureID, Status: football.FixtureStatus{Short: "FT"}},
			}, nil
		},
	}

	var finalizedMatchID, finalizedFixtureID int64
	called := false
	syncService := &mocks.MockMatchResultSyncService{
		FinalizeFunc: func(_ context.Context, matchID, fixtureID int64, _ *football.FixtureResponse) (*domain.SyncGroupStageOutcomes, error) {
			called = true
			finalizedMatchID = matchID
			finalizedFixtureID = fixtureID
			return &domain.SyncGroupStageOutcomes{}, nil
		},
	}

	job := &SyncMatchResultsJob{
		fetcher:     fetcher,
		syncService: syncService,
		logger:      &mocks.MockLogger{},
		timing:      neverEndingTiming(),
		timers:      make(map[int64]*time.Timer),
	}
	job.schedulePoll(1, 10, time.Now(), time.Hour) // pre-arm so we can prove it gets cleared

	job.poll(context.Background(), 1, 10, time.Now())

	if !called {
		t.Fatal("expected Finalize to be called for a finished fixture")
	}
	if finalizedMatchID != 1 || finalizedFixtureID != 10 {
		t.Fatalf("unexpected finalize args: match=%d fixture=%d", finalizedMatchID, finalizedFixtureID)
	}
	if _, ok := job.timers[1]; ok {
		t.Fatal("expected timer to be cleared after the match went final")
	}
}

// TestPoll_ReschedulesOnFetchError is the regression test for the abandon-on-error
// bug: a transient fetch failure must re-arm the poll, not drop the match.
func TestPoll_ReschedulesOnFetchError(t *testing.T) {
	fetcher := &mocks.MockFixtureFetcher{
		GetFixtureFunc: func(_ context.Context, _ int64) (*football.FixtureResponse, error) {
			return nil, errors.New("network blip")
		},
	}

	job := &SyncMatchResultsJob{
		fetcher: fetcher,
		logger:  &mocks.MockLogger{},
		timing:  neverEndingTiming(),
		timers:  make(map[int64]*time.Timer),
	}

	job.poll(context.Background(), 1, 10, time.Now())

	if _, ok := job.timers[1]; !ok {
		t.Fatal("expected a rescheduled timer after a fetch error (match must not be abandoned)")
	}
}

// TestPoll_ReschedulesWhileInProgress verifies a live (non-final) fixture re-arms
// the poll and does not persist a result.
func TestPoll_ReschedulesWhileInProgress(t *testing.T) {
	elapsed := 50
	fetcher := &mocks.MockFixtureFetcher{
		GetFixtureFunc: func(_ context.Context, fixtureID int64) (*football.FixtureResponse, error) {
			return &football.FixtureResponse{
				Fixture: football.FixtureInfo{ID: fixtureID, Status: football.FixtureStatus{Short: "2H", Elapsed: &elapsed}},
			}, nil
		},
	}

	job := &SyncMatchResultsJob{
		fetcher:     fetcher,
		syncService: &mocks.MockMatchResultSyncService{}, // Finalize must NOT be called (would panic)
		logger:      &mocks.MockLogger{},
		timing:      neverEndingTiming(),
		timers:      make(map[int64]*time.Timer),
	}

	job.poll(context.Background(), 1, 10, time.Now())

	if _, ok := job.timers[1]; !ok {
		t.Fatal("expected the poll to re-arm while the match is in progress")
	}
}

// TestPoll_StopsOnTerminalNotPlayed verifies a postponed/cancelled match stops
// polling and clears its timer.
func TestPoll_StopsOnTerminalNotPlayed(t *testing.T) {
	fetcher := &mocks.MockFixtureFetcher{
		GetFixtureFunc: func(_ context.Context, fixtureID int64) (*football.FixtureResponse, error) {
			return &football.FixtureResponse{
				Fixture: football.FixtureInfo{ID: fixtureID, Status: football.FixtureStatus{Short: "CANC"}},
			}, nil
		},
	}

	job := &SyncMatchResultsJob{
		fetcher: fetcher,
		logger:  &mocks.MockLogger{},
		timing:  neverEndingTiming(),
		timers:  make(map[int64]*time.Timer),
	}
	job.schedulePoll(1, 10, time.Now(), time.Hour) // pre-arm

	job.poll(context.Background(), 1, 10, time.Now())

	if _, ok := job.timers[1]; ok {
		t.Fatal("expected the timer to be cleared on a terminal non-played status")
	}
}

// TestPoll_GivesUpAfterMaxPollWindow verifies the global guard: past the window
// the poll gives up without even fetching.
func TestPoll_GivesUpAfterMaxPollWindow(t *testing.T) {
	job := &SyncMatchResultsJob{
		fetcher: &mocks.MockFixtureFetcher{}, // GetFixture must NOT be called (would panic)
		logger:  &mocks.MockLogger{},
		timing:  SyncTiming{MaxPollWindow: time.Minute},
		timers:  make(map[int64]*time.Timer),
	}

	kickoff := time.Now().Add(-2 * time.Hour) // well past kickoff + MaxPollWindow
	job.poll(context.Background(), 1, 10, kickoff)

	if _, ok := job.timers[1]; ok {
		t.Fatal("expected no timer to be armed past the poll window")
	}
}
