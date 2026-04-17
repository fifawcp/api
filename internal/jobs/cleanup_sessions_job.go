package jobs

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

type CleanupSessionsJob struct {
	sessionRepository domain.SessionRepository
	logger            logging.Logger
}

func NewCleanupSessionsJob(sessionRepository domain.SessionRepository, logger logging.Logger) *CleanupSessionsJob {
	return &CleanupSessionsJob{
		sessionRepository: sessionRepository,
		logger:            logger,
	}
}

func (j *CleanupSessionsJob) Name() string {
	return "cleanup:expired_sessions"
}

func (j *CleanupSessionsJob) Run(ctx context.Context) error {
	count, err := j.sessionRepository.DeleteExpiredSessions(ctx)
	if err != nil {
		j.logger.Error("failed to delete expired sessions", "error", err)
		return err
	}

	j.logger.Info("deleted expired sessions", "count", count)
	return nil
}
