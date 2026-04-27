package logging

import (
	"context"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/go-chi/chi/v5/middleware"
)

type slogAuditLogger struct {
	logger Logger
}

func NewAuditLogger(logger Logger) AuditLogger {
	return &slogAuditLogger{logger: logger}
}

func (a *slogAuditLogger) LogEvent(ctx context.Context, event Event) {
	fields := []any{
		LogName, "audit",
		RequestID, middleware.GetReqID(ctx),
		Action, string(event.Action),
		Resource, string(event.Resource),
		ResourceID, event.ResourceID,
		Outcome, string(event.Outcome),
	}

	if user := httpctx.GetAuthenticatedUser(ctx); user != nil {
		fields = append(fields, ActorID, user.ID, ActorRole, string(user.Role))
	}

	if info := httpctx.GetRequestInfo(ctx); info != nil {
		fields = append(fields, IP, info.IPAddress)
	}

	for k, v := range event.Metadata {
		fields = append(fields, k, v)
	}

	a.logger.Info("audit event", fields...)
}
