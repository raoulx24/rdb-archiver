// Package logging provides a minimal logging abstraction for rdb-archiver.
package logging

import (
	"fmt"
	"log/slog"
	"os"
)

// Logger defines the logging interface used across the application.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// SlogLogger implements Logger using Go 1.21+ slog for structured logging.
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a new SlogLogger with specified level and JSON/text output.
func NewSlogLogger(level string, format string) *SlogLogger {
	var lvl slog.Level

	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
	}

	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "text":
		// Custom text handler with level prefixes
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     lvl,
			AddSource: false,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					// wrap the string in slog.Value
					a.Value = slog.StringValue(fmt.Sprintf("[%s]", a.Value.String()))
				}
				return a
			},
		})
	default:
		// fallback to text
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &SlogLogger{
		logger: slog.New(handler),
	}
}

// Debug logs a debug message with optional key/value fields.
func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info logs an informational message with optional key/value fields.
func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn logs a warning message with optional key/value fields.
func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error logs an error message with optional key/value fields.
func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}
