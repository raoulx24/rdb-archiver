package main

import (
	"context"
	"path/filepath"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/snapshotwatcher"
	"github.com/raoulx24/rdb-archiver/internal/watchfs"
	"github.com/raoulx24/rdb-archiver/internal/worker"
)

// startConfigReload watches config.yaml and applies updates.
func startConfigReload(
	ctx context.Context,
	fw *watchfs.FileWatcher,
	sw *snapshotwatcher.SnapshotWatcher,
	wkr *worker.Worker,
	logg logging.Logger,
	configFile string,
	method string,
) {
	dir := filepath.Dir(configFile)
	base := filepath.Base(configFile)

	// Channel for config reload events
	reloadCh := make(chan struct{}, 1)

	go func() {
		if err := fw.StartWatchingForFile(ctx, method, dir, base, reloadCh, logg); err != nil {
			logg.Error("config watcher failed", "error", err)
		}
	}()

	// Debounce + reload logic
	var t *time.Timer
	for {
		select {
		case <-ctx.Done():
			return

		case <-reloadCh:
			if t != nil {
				t.Stop()
			}
			t = time.AfterFunc(300*time.Millisecond, func() {
				newCfg, err := config.Load(configFile)
				if err != nil {
					logg.Error("config reload failed", "error", err)
					return
				}

				fw.UpdateConfig(newCfg.WatchFS)
				sw.UpdateConfig(newCfg.Source)
				wkr.UpdateConfig(newCfg.Destination)

				logg.Info("config reloaded")
			})
		}
	}
}
