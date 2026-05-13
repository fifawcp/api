package mocks

import (
	"github.com/fifawcp/api/internal/domain"
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
	panic("RegisterJob called unexpectedly")
}

func (m *MockScheduler) Start() {
	if m.StartFunc != nil {
		m.StartFunc()
		return
	}
	panic("Start called unexpectedly")
}

func (m *MockScheduler) Stop() {
	if m.StopFunc != nil {
		m.StopFunc()
		return
	}
	panic("Stop called unexpectedly")
}
