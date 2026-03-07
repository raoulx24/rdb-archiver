package snapshotwatcher

import (
	"os"
	"path/filepath"

	"github.com/raoulx24/rdb-archiver/internal/snapshot"
)

// checkForNewSnapshot checks for a new snapshot and emits a job if needed.
func (sw *Watcher) checkForNewSnapshot() {
	sw.mu.RLock()
	dir := sw.cfg.Path
	primary := sw.cfg.PrimaryName
	aux := append([]string(nil), sw.cfg.AuxNames...)
	sw.mu.RUnlock()

	path := filepath.Join(dir, primary)

	info, err := os.Stat(path)
	if err != nil {
		return // file missing or unreadable
	}

	mod := info.ModTime()

	snap := snapshot.Snapshot{
		Dir:     dir,
		Primary: snapshot.FromFileInfo(path, info),
		Aux:     sw.loadAux(dir, aux),
	}

	sw.mu.Lock()
	sw.lastModTime = mod
	sw.mu.Unlock()

	sw.logg.Info("snapshot detected", "path", path)
	sw.mb.Put(snapshot.Job{Snap: snap})
}
