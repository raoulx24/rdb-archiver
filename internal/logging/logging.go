// Package logging provides a minimal logging abstraction for rdb-archiver.
package logging

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// Logger defines the logging interface used across the application.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// SlogLogger implements Logger using Go 1.21+ slog for structured logging.
type SlogLogger struct {
	mu     sync.RWMutex
	logger *slog.Logger
}

// NewSlogLogger creates a new SlogLogger with specified level and JSON/text output.
func NewSlogLogger(cfg Config) *SlogLogger {
	l := &SlogLogger{}
	l.applyConfig(cfg)
	return l
}

// UpdateConfig rebuilds the logger with new level/format settings.
func (l *SlogLogger) UpdateConfig(cfg Config) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.applyConfig(cfg)
}

// Debug logs a debug message with optional key/value fields.
func (l *SlogLogger) Debug(msg string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Debug(msg, args...)
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Info(msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Warn(msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Error(msg, args...)
}

func (l *SlogLogger) applyConfig(cfg Config) {
	var lvl slog.Level

	switch cfg.Level {
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
	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)

	case "text":
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     lvl,
			AddSource: false,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					a.Value = slog.StringValue(fmt.Sprintf("[%s]", a.Value.String()))
				}
				return a
			},
		})

	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	l.logger = slog.New(handler)
}
