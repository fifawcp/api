package logging

import "context"

type AuditLogger interface {
	LogEvent(ctx context.Context, event Event)
}

type Event struct {
	Action     AuditAction
	Resource   AuditResource
	ResourceID string
	Outcome    AuditOutcome
	Metadata   map[string]any // additional context (e.g. changed field values)
}

type AuditAction string

const (
	ActionUpdateMatchResult    AuditAction = "match.update_result"
	ActionResetMatchResult     AuditAction = "match.reset_result"
	ActionBulkUpdateMatches    AuditAction = "match.bulk_update_results"
	ActionRecalculateStandings AuditAction = "standing.recalculate"
	ActionResolveThirdPlace    AuditAction = "match.resolve_third_place"
	ActionRecordAwardWinners   AuditAction = "award.record_winners"
)

type AuditResource string

const (
	ResourceMatch    AuditResource = "match"
	ResourceStanding AuditResource = "standing"
	ResourceAward    AuditResource = "award"
)

type AuditOutcome string

const (
	OutcomeSuccess AuditOutcome = "success"
	OutcomeFailure AuditOutcome = "failure"
)
