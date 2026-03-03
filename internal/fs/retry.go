package fs

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// Operation describes a filesystem operation being retried
// Additional fields (paths, backend info, etc.) can be added later
type Operation struct {
	Name string
}

// RetryError reports that a transient operation exhausted all retries
type RetryError struct {
	Op  Operation
	Err error
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("%s failed after retries: %v", e.Op.Name, e.Err)
}

func (e *RetryError) Unwrap() error { return e.Err }

// retry executes fn with exponential backoff and jitter
// Transient errors are retried up to maxRetries. Permanent errors fail immediately
func retry(ctx context.Context, cfg Config, op Operation, fn func() error) error {
	maxRetries := cfg.MaxRetries
	base, err := time.ParseDuration(cfg.RetryBase)
	if err != nil {
		base = 50 * time.Millisecond
	}
	sleepCap, err := time.ParseDuration(cfg.RetryDurationCap)
	if err != nil {
		sleepCap = time.Second
	}

	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Abort early if the context is already done
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Permanent errors fail immediately
		if !isTransient(err) {
			return fmt.Errorf("%s failed permanently: %w", op.Name, err)
		}

		// No more attempts left
		if attempt == maxRetries {
			break
		}

		// Exponential backoff
		sleep := base * (1 << (attempt - 1))

		// cap at 1 second
		if sleep > sleepCap {
			sleep = sleepCap
		}

		// Add jitter: up to 50% extra delay
		jitter := time.Duration(rand.Int63n(int64(sleep / 2)))
		sleep += jitter

		// Sleep, but allow cancellation during the wait
		select {
		case <-time.After(sleep):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return &RetryError{Op: op, Err: lastErr}
}
