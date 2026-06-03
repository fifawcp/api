package logging

const (
	RequestID  = "request_id"  // chi request correlation ID
	UserID     = "user_id"     // authenticated actor
	Method     = "method"      // HTTP method
	Path       = "path"        // HTTP path
	Status     = "status"      // HTTP status code
	DurationMS = "duration_ms" // duration in milliseconds
	IP         = "ip"          // client IP address
	Error      = "error"       // error message
	Outcome    = "outcome"     // "success" | "user_error" | "server_error"
)

const (
	SessionID        = "session_id"
	RefreshOutcome   = "refresh_outcome" // success | rotated_past_grace | not_found_or_expired
	RefreshSource    = "refresh_source"  // client | middleware
	RotatedAgeMS     = "rotated_age_ms"  // small == concurrent race
	GraceWindowMS    = "grace_window_ms"
	TokenAgeMS       = "token_age_ms"
	TokenFingerprint = "token_fp"
)

// Audit-specific fields
const (
	LogName    = "log_name"    // always "audit" — filters audit vs operational logs
	ActorID    = "actor_id"    // user ID of the admin who performed the action
	Action     = "action"      // dotted action name, e.g. "match.update_result"
	Resource   = "resource"    // entity type, e.g. "match", "standing"
	ResourceID = "resource_id" // entity ID the action targeted
)
