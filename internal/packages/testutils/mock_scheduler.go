package testutils

import (
	"github.com/ncondes/fifawcp/internal/domain"
)

type MockScheduler struct {
	RegisterJobFunc func(spec string, job domain.Job) error
	StartFunc       func()
	StopFunc        func()
}

func (m *MockScheduler) RegisterJob(spec string, job domain.Job) error {
	if m.RegisterJobFunc != nil {
		return m.RegisterJobFunc(spec, job)
	}

	return nil
}

func (m *MockScheduler) Start() {
	if m.StartFunc != nil {
		m.StartFunc()
	}
}

func (m *MockScheduler) Stop() {
	if m.StopFunc != nil {
		m.StopFunc()
	}
}
