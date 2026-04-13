package scheduler

import "github.com/ncondes/fifawcp/internal/domain"

type Scheduler interface {
	RegisterJob(spec string, job domain.Job) error
	Start()
	Stop()
}
