package logging

import "log"

// Provides a simple logger interface for the application

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type StdLogger struct{}

func (StdLogger) Info(msg string, args ...any)  { log.Printf("INFO: "+msg, args...) }
func (StdLogger) Error(msg string, args ...any) { log.Printf("ERROR: "+msg, args...) }
