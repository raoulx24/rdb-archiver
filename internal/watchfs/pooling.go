package watchfs

import (
	"context"
	"time"
)

// WatchPolling emits events at a fixed interval.
func (w *FileWatcher) WatchPolling(
	ctx context.Context,
	events chan<- struct{},
) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			events <- struct{}{}
		}
	}
}
