package fs

import (
	"context"
	"fmt"
	"time"
)

// implements retry logic with exponential backoff.
// It is used by copy and rename operations to handle transient filesystem errors.

func retry(ctx context.Context, opName string, fn func() error) error {
	const maxRetries = 5
	base := 100 * time.Millisecond

	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if !isTransient(err) {
			return fmt.Errorf("%s failed permanently: %w", opName, err)
		}

		if attempt == maxRetries {
			break
		}

		sleep := base * (1 << (attempt - 1))
		time.Sleep(sleep)
	}

	return fmt.Errorf("%s failed after %d retries: %w", opName, maxRetries, lastErr)
}
