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

	reloadCh := make(chan struct{}, 1)

	// watcher cancel function for config.yaml watcher
	var watchCancel context.CancelFunc

	startWatcher := func(mode string) {
		if watchCancel != nil {
			watchCancel()
		}

		var wctx context.Context
		wctx, watchCancel = context.WithCancel(ctx)

		go func() {
			if err := fw.StartWatchingForFile(wctx, mode, dir, base, reloadCh, logg); err != nil {
				logg.Error("config watcher failed", "error", err)
			}
		}()
	}

	// start initial config watcher
	startWatcher(method)

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

				// update file watcher config
				if err := fw.UpdateConfig(newCfg.WatchFS); err != nil {
					logg.Error("file watcher config update failed", "error", err)
				}

				// snapshot watcher: detect if restart needed
				oldSnapCfg := sw.CurrentConfig()
				sw.UpdateConfig(newCfg.Source)
				if sw.NeedsRestart(oldSnapCfg, newCfg.Source) {
					startSnapshotWatcher(ctx, sw, logg)
				}

				// worker config
				wkr.UpdateConfig(newCfg.Destination)

				// restart config watcher if its method changed
				if newCfg.ConfigReload.Method != method {
					method = newCfg.ConfigReload.Method
					startWatcher(method)
				}

				logg.Info("config reloaded")
			})
		}
	}
}
