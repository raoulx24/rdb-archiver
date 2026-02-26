package watcher

import (
	"context"
	"time"
)

// StartPolling triggers detect() on a fixed interval.
func (w *Watcher) StartPolling(ctx context.Context) {
	w.mu.RLock()
	interval := w.interval
	w.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.detect()
		}
	}
}
