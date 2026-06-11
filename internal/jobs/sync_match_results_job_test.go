package jobs

import (
	"testing"
	"time"
)

// TestScheduleSync_ReplacesExistingTimer verifies that re-scheduling the same
// match stops the previous timer and keeps exactly one pending timer, so daily
// re-runs do not accumulate duplicate timers.
func TestScheduleSync_ReplacesExistingTimer(t *testing.T) {
	job := &SyncMatchResultsJob{timers: make(map[int64]*time.Timer)}

	const matchID int64 = 1
	const fixtureID int64 = 10

	// A long delay guarantees neither timer fires during the test.
	job.scheduleSync(matchID, fixtureID, time.Hour)
	firstTimer := job.timers[matchID]

	job.scheduleSync(matchID, fixtureID, time.Hour)
	secondTimer := job.timers[matchID]

	if len(job.timers) != 1 {
		t.Fatalf("expected exactly 1 pending timer, got %d", len(job.timers))
	}
	if firstTimer == secondTimer {
		t.Fatal("expected the timer to be replaced on re-schedule")
	}
	// The first timer should already be stopped by the helper, so Stop() reports false.
	if firstTimer.Stop() {
		t.Fatal("expected the previous timer to have been stopped")
	}
}

// TestStop_CancelsAllPendingTimers verifies shutdown cancels every pending timer.
func TestStop_CancelsAllPendingTimers(t *testing.T) {
	job := &SyncMatchResultsJob{timers: make(map[int64]*time.Timer)}

	job.scheduleSync(1, 10, time.Hour)
	job.scheduleSync(2, 20, time.Hour)
	timer := job.timers[1]

	job.Stop()

	if len(job.timers) != 0 {
		t.Fatalf("expected all timers cleared, got %d", len(job.timers))
	}
	if timer.Stop() {
		t.Fatal("expected pending timer to have been stopped")
	}
}
