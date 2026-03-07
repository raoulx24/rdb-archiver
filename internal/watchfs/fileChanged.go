package watchfs

import "os"

// isWatchedFileChanged checks if the file has been modified since the last event
func (wfs *FileWatcher) isWatchedFileChanged(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	mod := info.ModTime()

	wfs.mu.Lock()
	defer wfs.mu.Unlock()

	last, ok := wfs.lastModTime[path]
	if ok && !mod.After(last) {
		return false
	}

	wfs.lastModTime[path] = mod
	return true
}
