package watcher

import (
	"os"
	"path/filepath"

	"github.com/raoulx24/rdb-archiver/internal/snapshot"
	"github.com/raoulx24/rdb-archiver/internal/worker"
)

// detect builds and enqueues a snapshot if the primary file changed.
func (w *Watcher) detect() {
	w.mu.RLock()
	dir := w.dir
	primary := w.primaryName
	aux := append([]string(nil), w.auxNames...)
	last := w.lastModTime
	w.mu.RUnlock()

	path := filepath.Join(dir, primary)

	info, err := os.Stat(path)
	if err != nil {
		return
	}

	mod := info.ModTime()
	if !mod.After(last) {
		return
	}

	snap := snapshot.Snapshot{
		Dir:     dir,
		Primary: snapshot.FromFileInfo(path, info),
		Aux:     w.loadAux(dir, aux),
	}

	w.mu.Lock()
	w.lastModTime = mod
	w.mu.Unlock()

	w.mb.Put(worker.Job{Snap: snap})
}
