package scheduler

import (
	"context"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/robfig/cron/v3"
)

type CronScheduler struct {
	cron   *cron.Cron
	logger logging.Logger
}

func NewCronScheduler(logger logging.Logger) *CronScheduler {
	return &CronScheduler{
		cron:   cron.New(),
		logger: logger,
	}
}

func (c *CronScheduler) RegisterJob(spec string, job domain.Job) error {
	// AddFunc schedules a function to run on the cron expression (spec).
	// It returns an EntryID (ignored here) and an error if the spec is invalid.
	_, err := c.cron.AddFunc(spec, func() {
		// This anonymous function is what cron calls on every tick.
		// It wraps the job with logging and safety guards.
		c.logger.Info("starting job", "job", job.Name())
		start := time.Now()

		// Panic recovery: if the job panics (e.g. nil pointer), we catch it here
		// so the cron scheduler itself doesn't crash and keeps running future ticks.
		defer func() {
			if r := recover(); r != nil {
				c.logger.Error("job panicked", "job", job.Name(), "panic", r)
			}
		}()

		// Run the actual job logic. context.Background() is used because cron
		// jobs have no incoming request context — they originate from the scheduler.
		if err := job.Run(context.Background()); err != nil {
			c.logger.Error("job failed", "job", job.Name(), "error", err, "duration", time.Since(start))
			return
		}

		c.logger.Info("job completed", "job", job.Name(), "duration", time.Since(start))
	})

	// Returns non-nil if the cron expression is invalid (e.g. "not a valid spec").
	// This is checked at startup so misconfigured schedules fail fast.
	return err
}

func (c *CronScheduler) Start() {
	c.cron.Start()
}

func (c *CronScheduler) Stop() {
	ctx := c.cron.Stop()
	<-ctx.Done() // Blocks until all running jobs are complete
}
