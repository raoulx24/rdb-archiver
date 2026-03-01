package watchfs

import (
	"os"
	"path/filepath"
	"time"
)

func (fsw *FileWatcher) isPrimaryFileStable(dir, file string) bool {
	fsw.mu.RLock()
	stability := fsw.stabilityWindow
	fsw.mu.RUnlock()

	path := filepath.Join(dir, file)

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
