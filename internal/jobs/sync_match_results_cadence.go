package jobs

import "time"

// api-football live status short codes.
// See https://www.api-football.com/documentation-v3#tag/Fixtures
const (
	statusNotStarted      = "NS"
	statusTimeToBeDefined = "TBD"
	statusFirstHalf       = "1H"
	statusHalfTime        = "HT"
	statusSecondHalf      = "2H"
	statusExtraTime       = "ET"
	statusBreakTime       = "BT"
	statusPenaltiesLive   = "P"
	statusSuspended       = "SUSP"
	statusInterrupted     = "INT"
)

// terminalNotPlayedStatuses will never produce a result, so the poller gives up
// instead of waiting out the poll window.
var terminalNotPlayedStatuses = map[string]bool{
	"PST":  true, // Postponed
	"CANC": true, // Cancelled
	"ABD":  true, // Abandoned
	"WO":   true, // WalkOver
	"AWD":  true, // Technical loss / awarded
}

const (
	// In the second half, switch from a time-based estimate to the tight
	// near-end cadence once the match is within stoppage-time range.
	secondHalfNearEndElapsed = 80
	firstHalfFullElapsed     = 45
	secondHalfFullElapsed    = 90

	// Coarse re-check gaps for states still far from a result.
	firstHalfToFullTimeGap = 20 * time.Minute // ~half time (15m) + cushion to reach full time
	halfTimeRecheck        = 10 * time.Minute
	kickoffSlippedRecheck  = 5 * time.Minute // NS/TBD past the first poll
	suspendedRecheck       = 5 * time.Minute // SUSP/INT
)

// pollDecision is the outcome of inspecting one live (non-final) fixture poll.
type pollDecision struct {
	stop  bool          // give up: terminal "not played" status
	delay time.Duration // otherwise, wait this long before the next poll
}

// nextPollDelay decides how long to wait before re-polling a match that is not
// yet final, from its live status and elapsed minute. It is pure (no clock, no
// I/O) so the cadence is exhaustively unit-tested. Final statuses (FT/AET/PEN)
// never reach here — the poller persists those before calling this.
//
// nearInterval and maxInterval are the clamp bounds; every estimated delay is
// kept within [nearInterval, maxInterval].
func nextPollDelay(short string, elapsed *int, nearInterval, maxInterval time.Duration) pollDecision {
	if terminalNotPlayedStatuses[short] {
		return pollDecision{stop: true}
	}

	clamp := func(delay time.Duration) time.Duration {
		switch {
		case delay < nearInterval:
			return nearInterval
		case delay > maxInterval:
			return maxInterval
		default:
			return delay
		}
	}

	switch short {
	// Suspended/interrupted: sparse re-check; the poll window eventually ends it.
	case statusSuspended, statusInterrupted:
		return pollDecision{delay: clamp(suspendedRecheck)}

	// Kickoff slipped (still not started past the first poll): short re-check.
	case statusNotStarted, statusTimeToBeDefined, "":
		return pollDecision{delay: clamp(kickoffSlippedRecheck)}

	case statusHalfTime:
		return pollDecision{delay: clamp(halfTimeRecheck)}

	// End of regulation or the knockout tail (extra time / penalties): poll tight.
	case statusExtraTime, statusBreakTime, statusPenaltiesLive:
		return pollDecision{delay: nearInterval}

	case statusSecondHalf:
		elapsedMinute := elapsedOr(elapsed, secondHalfFullElapsed)
		if elapsedMinute >= secondHalfNearEndElapsed {
			return pollDecision{delay: nearInterval}
		}
		untilNearEnd := time.Duration(secondHalfNearEndElapsed-elapsedMinute) * time.Minute
		return pollDecision{delay: clamp(untilNearEnd)}

	case statusFirstHalf:
		elapsedMinute := elapsedOr(elapsed, firstHalfFullElapsed)
		untilFullTime := time.Duration(firstHalfFullElapsed-elapsedMinute)*time.Minute + firstHalfToFullTimeGap
		return pollDecision{delay: clamp(untilFullTime)}

	// Unknown/unexpected live code: poll tight so a fast finish isn't missed.
	default:
		return pollDecision{delay: nearInterval}
	}
}

// elapsedOr returns the elapsed minute, or fallback when api-football omits it.
func elapsedOr(elapsed *int, fallback int) int {
	if elapsed == nil {
		return fallback
	}
	return *elapsed
}
