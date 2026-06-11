package jobs

import (
	"testing"
	"time"
)

func TestNextPollDelay(t *testing.T) {
	const near = 60 * time.Second
	const max = 15 * time.Minute

	intPtr := func(n int) *int { return &n }

	cases := []struct {
		name        string
		short       string
		elapsed     *int
		expectStop  bool
		expectDelay time.Duration
	}{
		// Second half: tight near the end, time-based estimate earlier (clamped).
		{"2H past 80'", statusSecondHalf, intPtr(82), false, near},
		{"2H at 80'", statusSecondHalf, intPtr(80), false, near},
		{"2H mid", statusSecondHalf, intPtr(70), false, 10 * time.Minute},
		{"2H early clamps to max", statusSecondHalf, intPtr(50), false, max},
		{"2H nil elapsed assumes full time", statusSecondHalf, nil, false, near},

		// Knockout tail is always tight.
		{"extra time", statusExtraTime, intPtr(95), false, near},
		{"break time", statusBreakTime, nil, false, near},
		{"penalties", statusPenaltiesLive, nil, false, near},

		// Mid-match coarse cadences.
		{"half time", statusHalfTime, nil, false, halfTimeRecheck},
		{"first half clamps to max", statusFirstHalf, intPtr(30), false, max},
		{"first half late still clamps", statusFirstHalf, intPtr(44), false, max},

		// Kickoff slipped / odd states.
		{"not started", statusNotStarted, nil, false, kickoffSlippedRecheck},
		{"to be defined", statusTimeToBeDefined, nil, false, kickoffSlippedRecheck},
		{"empty status", "", nil, false, kickoffSlippedRecheck},
		{"suspended", statusSuspended, intPtr(60), false, suspendedRecheck},
		{"interrupted", statusInterrupted, nil, false, suspendedRecheck},

		// Terminal, not played -> give up.
		{"postponed", "PST", nil, true, 0},
		{"cancelled", "CANC", nil, true, 0},
		{"abandoned", "ABD", nil, true, 0},
		{"walkover", "WO", nil, true, 0},
		{"awarded", "AWD", nil, true, 0},

		// Unknown live code -> fail fast (tight).
		{"unknown", "XYZ", nil, false, near},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nextPollDelay(tc.short, tc.elapsed, near, max)
			if got.stop != tc.expectStop {
				t.Fatalf("stop = %v, want %v", got.stop, tc.expectStop)
			}
			if !tc.expectStop && got.delay != tc.expectDelay {
				t.Fatalf("delay = %v, want %v", got.delay, tc.expectDelay)
			}
		})
	}
}
