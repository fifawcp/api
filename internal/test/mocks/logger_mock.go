package mocks

import (
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

type MockLogger struct {
	DebugFunc func(msg string, args ...any)
	InfoFunc  func(msg string, args ...any)
	WarnFunc  func(msg string, args ...any)
	ErrorFunc func(msg string, args ...any)
	FatalFunc func(msg string, args ...any)
	WithFunc  func(args ...any) logging.Logger
}

func NewMockLogger(
	DebugFunc func(msg string, args ...any),
	InfoFunc func(msg string, args ...any),
	WarnFunc func(msg string, args ...any),
	ErrorFunc func(msg string, args ...any),
	FatalFunc func(msg string, args ...any),
	WithFunc func(args ...any) logging.Logger,
) *MockLogger {
	return &MockLogger{
		DebugFunc: DebugFunc,
		InfoFunc:  InfoFunc,
		WarnFunc:  WarnFunc,
		ErrorFunc: ErrorFunc,
		FatalFunc: FatalFunc,
		WithFunc:  WithFunc,
	}
}

func (l *MockLogger) Debug(msg string, args ...any) {
	if l.DebugFunc != nil {
		l.DebugFunc(msg, args...)
	}
}

func (l *MockLogger) Info(msg string, args ...any) {
	if l.InfoFunc != nil {
		l.InfoFunc(msg, args...)
	}
}

func (l *MockLogger) Warn(msg string, args ...any) {
	if l.WarnFunc != nil {
		l.WarnFunc(msg, args...)
	}
}

func (l *MockLogger) Error(msg string, args ...any) {
	if l.ErrorFunc != nil {
		l.ErrorFunc(msg, args...)
	}
}

func (l *MockLogger) Fatal(msg string, args ...any) {
	if l.FatalFunc != nil {
		l.FatalFunc(msg, args...)
	}
}

func (l *MockLogger) With(args ...any) logging.Logger {
	if l.WithFunc != nil {
		return l.WithFunc(args...)
	}
	return l
}
