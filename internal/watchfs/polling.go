package watchfs

import (
	"context"
	"path/filepath"
	"time"
)

// WatchPolling emits events at a fixed interval.
func (wfs *FileWatcher) WatchPolling(
	ctx context.Context,
	dir string,
	file string,
	events chan<- struct{},
) error {
	wfs.logg.Info("starting watch fs - polling mode", "pollInterval", wfs.pollInterval)
	ticker := time.NewTicker(wfs.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if wfs.isWatchedFileChanged(filepath.Join(dir, file)) {
				events <- struct{}{}
			}
		}
	}
}
