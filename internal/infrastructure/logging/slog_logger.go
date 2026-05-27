package logging

import (
	"log/slog"
	"os"

	"github.com/fifawcp/api/internal/infrastructure/config"
)

type SlogLogger struct {
	logger      *slog.Logger
	rootHandler slog.Handler   // raw handler before any With chains, used by WithReplace
	baseFields  map[string]any // tracks all accumulated fields for WithReplace merging
}

func NewSlogLogger(cfg *config.Config) *SlogLogger {
	replaceAttr := func(_ []string, attr slog.Attr) slog.Attr {
		switch attr.Key {
		case slog.LevelKey:
			attr.Key = "severity"
			if level, ok := attr.Value.Any().(slog.Level); ok {
				switch {
				case level >= slog.LevelError:
					attr.Value = slog.StringValue("ERROR")
				case level >= slog.LevelWarn:
					attr.Value = slog.StringValue("WARNING")
				case level >= slog.LevelInfo:
					attr.Value = slog.StringValue("INFO")
				default:
					attr.Value = slog.StringValue("DEBUG")
				}
			}
		case slog.MessageKey:
			attr.Key = "message"
		}

		return attr
	}

	options := &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: replaceAttr,
	}

	var rootHandler slog.Handler
	if cfg.IsProd() {
		rootHandler = slog.NewJSONHandler(os.Stdout, options)
	} else {
		options.Level = slog.LevelDebug
		rootHandler = slog.NewTextHandler(os.Stdout, options)
	}

	base := map[string]any{
		"service": "fifawcp-api",
		"env":     cfg.Env,
	}

	return &SlogLogger{
		logger:      buildLogger(rootHandler, base),
		rootHandler: rootHandler,
		baseFields:  base,
	}
}

func buildLogger(root slog.Handler, fields map[string]any) *slog.Logger {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}

	return slog.New(root).With(attrs...)
}

// parseKVArgs converts a flat key-value ...any slice into a map.
// Odd-count slices silently drop the trailing key.
func parseKVArgs(args []any) map[string]any {
	m := make(map[string]any, len(args)/2)
	for i := 0; i+1 < len(args); i += 2 {
		if key, ok := args[i].(string); ok {
			m[key] = args[i+1]
		}
	}
	return m
}

func (l *SlogLogger) Debug(msg string, args ...any) { l.logger.Debug(msg, args...) }
func (l *SlogLogger) Info(msg string, args ...any)  { l.logger.Info(msg, args...) }
func (l *SlogLogger) Warn(msg string, args ...any)  { l.logger.Warn(msg, args...) }
func (l *SlogLogger) Error(msg string, args ...any) { l.logger.Error(msg, args...) }
func (l *SlogLogger) Fatal(msg string, args ...any) {
	l.logger.Error(msg, args...)
	os.Exit(1)
}

func (l *SlogLogger) With(args ...any) Logger {
	newBase := make(map[string]any, len(l.baseFields)+len(args)/2)
	for k, v := range l.baseFields {
		newBase[k] = v
	}

	for k, v := range parseKVArgs(args) {
		newBase[k] = v
	}

	return &SlogLogger{
		logger:      l.logger.With(args...),
		rootHandler: l.rootHandler,
		baseFields:  newBase,
	}
}
