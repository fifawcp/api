package scheduler

import "github.com/fifawcp/api/internal/domain"

type Scheduler interface {
	RegisterJob(spec string, job domain.Job) error
	Start()
	Stop()
}
