package watcher

import (
	"os"
	"path/filepath"
	"time"
)

func (w *Watcher) isPrimaryFileStable() bool {
	w.mu.RLock()
	path := filepath.Join(w.dir, w.primaryName)
	stability := w.stability
	w.mu.RUnlock()

	info1, err := os.Stat(path)
	if err != nil {
		return false
	}

	size1 := info1.Size()

	time.Sleep(stability)

	info2, err := os.Stat(path)
	if err != nil {
		return false
	}

	size2 := info2.Size()

	return size1 == size2
}
