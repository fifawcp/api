package jobs

import (
	"context"

	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/services"
)

type SyncMatchResultsJob struct {
	matchService services.MatchServiceInterface
	logger       logging.Logger
}

func NewSyncMatchResultsJob(
	matchService services.MatchServiceInterface,
	logger logging.Logger,
) *SyncMatchResultsJob {
	return &SyncMatchResultsJob{
		matchService: matchService,
		logger:       logger,
	}
}

func (j *SyncMatchResultsJob) Name() string {
	return "sync:match_results"
}

func (j *SyncMatchResultsJob) Run(ctx context.Context) error {
	j.logger.Info("sync:match_results started")

	// TODO: fetch latest results from data provider API
	// 1. Get results from data provider API
	// 2. Map results to domain.MatchResultUpdate
	// 3. Call UpdateMatchResultsBulk

	j.logger.Info("sync:match_results completed")
	return nil
}
