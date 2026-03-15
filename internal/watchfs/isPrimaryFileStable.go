package watchfs

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// isPrimaryFileStable checks if the primary file is stable by comparing its size before and after a stability window.
func (wfs *FileWatcher) isPrimaryFileStable(ctx context.Context, dir, file string) bool {
	wfs.mu.RLock()
	stability := wfs.stabilityWindow
	wfs.mu.RUnlock()

	path := filepath.Join(dir, file)

	info1, err := os.Stat(path)
	if err != nil {
		wfs.logg.Error("failed to stat file", "path", path, "error", err)
		return false
	}
	size1 := info1.Size()

	// Use a timer instead of time.Sleep
	timer := time.NewTimer(stability)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		wfs.logg.Debug("file stability check cancelled", "function", "isPrimaryFileStable", "path", path)
		// Caller cancelled the check
		return false
	case <-timer.C:
		wfs.logg.Debug("stability window passed", "function", "isPrimaryFileStable", "path", path)
		// Stability window passed
	}

	info2, err := os.Stat(path)
	if err != nil {
		wfs.logg.Error("failed to stat file", "path", path, "error", err)
		return false
	}
	size2 := info2.Size()
	wfs.logg.Debug("file sizes compared", "function", "isPrimaryFileStable", "path", path, "lastSize", size1, "size", size2)

	return size1 == size2
}
