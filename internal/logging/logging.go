// Package logging provides a minimal logging abstraction for rdb-archiver.
package logging

import (
	"context"
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
	With(args ...any) Logger
}

// SlogLogger implements Logger using Go 1.21+ slog for structured logging.
type SlogLogger struct {
	mu      *sync.RWMutex
	handler *slog.Handler
	attrs   []any
}

// NewSlogLogger creates a new SlogLogger with specified level and JSON/text output.
func NewSlogLogger(cfg Config) *SlogLogger {
	mu := &sync.RWMutex{}
	var handler slog.Handler

	l := &SlogLogger{
		mu:      mu,
		handler: &handler,
	}

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
	l.log(slog.LevelDebug, msg, args...)
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.log(slog.LevelInfo, msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.log(slog.LevelWarn, msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.log(slog.LevelError, msg, args...)
}

func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{
		mu:      l.mu,
		handler: l.handler,
		attrs:   append(append([]any{}, l.attrs...), args...),
	}
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

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	*l.handler = handler
}

func (l *SlogLogger) log(level slog.Level, msg string, args ...any) {
	l.mu.RLock()
	h := *l.handler
	attrs := append([]any{}, l.attrs...)
	l.mu.RUnlock()

	logger := slog.New(h).With(attrs...)
	logger.Log(context.Background(), level, msg, args...)
}
