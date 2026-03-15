package watchfs

import (
	"os"
	"time"
)

// isWatchedFileChanged checks if the file has been modified since the last event
func (wfs *FileWatcher) isWatchedFileChanged(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		wfs.logg.Error("failed to stat file", "path", path, "error", err)
		return false
	}

	mod := info.ModTime()

	wfs.mu.Lock()
	defer wfs.mu.Unlock()

	last, ok := wfs.lastModTime[path]
	if ok && !mod.After(last) {
		wfs.logg.Debug("file not modified since last event",
			"function", "isWatchedFileChanged", "path", path,
			"lastModifyTime", last.Format(time.RFC3339), "modifyTime", mod.Format(time.RFC3339))
		return false
	}

	wfs.lastModTime[path] = mod
	wfs.logg.Debug("file is modified since last event", "function", "isWatchedFileChanged", "path", path)
	return true
}
