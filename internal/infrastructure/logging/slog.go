package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/ncondes/fifawcp/internal/infrastructure/config"
)

type SlogLogger struct {
	logger *slog.Logger
}

func NewSlogLogger(config *config.Config) *SlogLogger {
	var handler slog.Handler

	options := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if config.IsProd() {
		handler = slog.NewJSONHandler(os.Stdout, options)
	} else {
		options.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, options)
	}

	return &SlogLogger{
		logger: slog.New(handler),
	}
}

func (l *SlogLogger) Debug(msg string, keysAndValues ...any) {
	l.logger.Debug(msg, keysAndValues...)
}

func (l *SlogLogger) Info(msg string, keysAndValues ...any) {
	l.logger.Info(msg, keysAndValues...)
}

func (l *SlogLogger) Warn(msg string, keysAndValues ...any) {
	l.logger.Warn(msg, keysAndValues...)
}

func (l *SlogLogger) Error(msg string, keysAndValues ...any) {
	l.logger.Error(msg, keysAndValues...)
}

func (l *SlogLogger) Fatal(msg string, keysAndValues ...any) {
	l.logger.Error(msg, keysAndValues...)
	os.Exit(1)
}

func (l *SlogLogger) With(keysAndValues ...any) Logger {
	return &SlogLogger{
		logger: l.logger.With(keysAndValues...),
	}
}

func (l *SlogLogger) WithContext(ctx context.Context) Logger {
	return &SlogLogger{
		logger: l.logger.With("context", ctx),
	}
}
