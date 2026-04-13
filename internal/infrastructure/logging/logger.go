package logging

import "context"

type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	Fatal(msg string, keysAndValues ...any)
	With(keysAndValues ...any) Logger
	WithContext(ctx context.Context) Logger
}
