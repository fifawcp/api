package jobs

import (
	"context"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

type CleanupSessionsJob struct {
	sessionRepository      domain.SessionRepository
	refreshTokenRepository domain.RefreshTokenRepository
	graceWindow            time.Duration
	logger                 logging.Logger
}

func NewCleanupSessionsJob(
	sessionRepository domain.SessionRepository,
	refreshTokenRepository domain.RefreshTokenRepository,
	graceWindow time.Duration,
	logger logging.Logger,
) *CleanupSessionsJob {
	return &CleanupSessionsJob{
		sessionRepository:      sessionRepository,
		refreshTokenRepository: refreshTokenRepository,
		graceWindow:            graceWindow,
		logger:                 logger,
	}
}

func (j *CleanupSessionsJob) Name() string {
	return "cleanup:expired_sessions"
}

func (j *CleanupSessionsJob) Run(ctx context.Context) error {
	count, err := j.sessionRepository.DeleteExpiredSessions(ctx)
	if err != nil {
		j.logger.Error(
			"failed to delete expired sessions",
			logging.Error, err.Error(),
		)
		return err
	}

	j.logger.Info(
		"deleted expired sessions",
		"count", count,
	)

	// Sweep rotated (superseded) refresh tokens whose grace window has elapsed.
	// Active sessions self-prune on rotation; this catches idle sessions' leftovers.
	rotated, err := j.refreshTokenRepository.DeleteRotatedBefore(ctx, time.Now().Add(-j.graceWindow))
	if err != nil {
		j.logger.Error(
			"failed to delete rotated refresh tokens",
			logging.Error, err.Error(),
		)
		return err
	}

	j.logger.Info(
		"deleted rotated refresh tokens",
		"count", rotated,
	)
	return nil
}
