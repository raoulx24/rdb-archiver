package snapshotwatcher

import (
	"os"
	"path/filepath"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/snapshot"
)

// checkForNewSnapshot checks for a new snapshot and emits a job if needed.
func (sw *Watcher) checkForNewSnapshot() {
	start := time.Now()
	defer sw.metrics.ObserveDetectionDuration(time.Since(start))

	sw.mu.RLock()
	dir := sw.cfg.Path
	primary := sw.cfg.PrimaryName
	aux := append([]string(nil), sw.cfg.AuxNames...)
	sw.mu.RUnlock()

	path := filepath.Join(dir, primary)
	sw.logg.Debug("checking for new snapshot", "function", "checkForNewSnapshot", "dir", dir, "primary", primary)

	info, err := os.Stat(path)
	if err != nil {
		sw.metrics.InvalidSnapshot()
		sw.logg.Debug("failed to stat snapshot file", "error", err)
		return // file missing or unreadable
	}

	mod := info.ModTime()

	snap := snapshot.Snapshot{
		Dir:     dir,
		Primary: snapshot.FromFileInfo(path, info),
		Aux:     sw.loadAux(dir, aux),
	}
	sw.metrics.SnapshotParsed()

	sw.mu.Lock()
	sw.lastModTime = mod
	sw.mu.Unlock()

	sw.logg.Info("snapshot detected", "path", path)
	sw.logg.Debug("sending snapshot job", "function", "checkForNewSnapshot")
	sw.mb.Put(snapshot.Job{Snap: snap})
	sw.metrics.JobEnqueued()
}
