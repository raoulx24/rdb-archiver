package watchfs

import (
	"context"
	"time"
)

// WatchPolling emits events at a fixed interval.
func (wfs *FileWatcher) WatchPolling(
	ctx context.Context,
	events chan<- struct{},
) error {
	wfs.logg.Info("starting watch fs - pooling mode", "poolInterval", wfs.pollInterval)
	ticker := time.NewTicker(wfs.pollInterval)
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
