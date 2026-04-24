package mocks

import (
	"context"

	"github.com/fifawcp/api/internal/infrastructure/logging"
)

type MockLogger struct {
	DebugFunc func(msg string, keysAndValues ...any)
	InfoFunc  func(msg string, keysAndValues ...any)
	WarnFunc  func(msg string, keysAndValues ...any)
	ErrorFunc func(msg string, keysAndValues ...any)
	FatalFunc func(msg string, keysAndValues ...any)
	WithFunc  func(keysAndValues ...any) logging.Logger
}

func NewMockLogger(
	DebugFunc func(msg string, keysAndValues ...any),
	InfoFunc func(msg string, keysAndValues ...any),
	WarnFunc func(msg string, keysAndValues ...any),
	ErrorFunc func(msg string, keysAndValues ...any),
	FatalFunc func(msg string, keysAndValues ...any),
	WithFunc func(keysAndValues ...any) logging.Logger,
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

func (l *MockLogger) Debug(msg string, keysAndValues ...any) {
	if l.DebugFunc != nil {
		l.DebugFunc(msg, keysAndValues...)
	}
}

func (l *MockLogger) Info(msg string, keysAndValues ...any) {
	if l.InfoFunc != nil {
		l.InfoFunc(msg, keysAndValues...)
	}
}

func (l *MockLogger) Warn(msg string, keysAndValues ...any) {
	if l.WarnFunc != nil {
		l.WarnFunc(msg, keysAndValues...)
	}
}

func (l *MockLogger) Error(msg string, keysAndValues ...any) {
	if l.ErrorFunc != nil {
		l.ErrorFunc(msg, keysAndValues...)
	}
}

func (l *MockLogger) Fatal(msg string, keysAndValues ...any) {
	if l.FatalFunc != nil {
		l.FatalFunc(msg, keysAndValues...)
	}
}

func (l *MockLogger) With(keysAndValues ...any) logging.Logger {
	if l.WithFunc != nil {
		return l.WithFunc(keysAndValues...)
	}
	return l
}

func (l *MockLogger) WithContext(ctx context.Context) logging.Logger {
	return l
}
