// Package logging provides a minimal logging abstraction for rdb-archiver.
package logging

import "log"

// Logger defines the logging interface used across the application.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// StdLogger implements Logger using the standard library log package.
type StdLogger struct{}

func (StdLogger) Info(msg string, args ...any) {
	log.Printf("[INFO]: "+msg, args...)
}

func (StdLogger) Warn(msg string, args ...any) {
	log.Printf("[WARN]: "+msg, args...)
}

func (StdLogger) Error(msg string, args ...any) {
	log.Printf("[ERR]: "+msg, args...)
}
