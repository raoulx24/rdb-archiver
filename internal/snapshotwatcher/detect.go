package snapshotwatcher

import (
	"os"
	"path/filepath"

	"github.com/raoulx24/rdb-archiver/internal/snapshot"
)

// checkForNewSnapshot checks for a new snapshot and emits a job if needed.
func (w *SnapshotWatcher) checkForNewSnapshot() {
	w.mu.RLock()
	dir := w.cfg.Path
	primary := w.cfg.PrimaryName
	aux := append([]string(nil), w.cfg.AuxNames...)
	last := w.lastModTime
	w.mu.RUnlock()

	path := filepath.Join(dir, primary)

	info, err := os.Stat(path)
	if err != nil {
		return // file missing or unreadable
	}

	mod := info.ModTime()
	if !mod.After(last) {
		return // no new snapshot
	}

	snap := snapshot.Snapshot{
		Dir:     dir,
		Primary: snapshot.FromFileInfo(path, info),
		Aux:     w.loadAux(dir, aux),
	}

	w.mu.Lock()
	w.lastModTime = mod
	w.mu.Unlock()

	w.log.Debug("snapshot detected", "path", path)
	w.mb.Put(snapshot.Job{Snap: snap})
}
