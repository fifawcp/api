package logging

import "context"

type NoopLogger struct{}

func NewNoopLogger() Logger {
	return &NoopLogger{}
}

func (l *NoopLogger) Debug(msg string, keysAndValues ...any) {}
func (l *NoopLogger) Info(msg string, keysAndValues ...any)  {}
func (l *NoopLogger) Warn(msg string, keysAndValues ...any)  {}
func (l *NoopLogger) Error(msg string, keysAndValues ...any) {}
func (l *NoopLogger) Fatal(msg string, keysAndValues ...any) {}
func (l *NoopLogger) With(keysAndValues ...any) Logger       { return l }
func (l *NoopLogger) WithContext(ctx context.Context) Logger { return l }
