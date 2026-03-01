package main

import (
	"context"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
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
) {
	configFile := "config.yaml"
	dir := filepath.Dir(configFile)
	base := filepath.Base(configFile)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		logg.Error("config watcher init failed", "error", err)
		return
	}
	defer w.Close()

	if err := w.Add(dir); err != nil {
		logg.Error("config watcher add failed", "error", err)
		return
	}

	resetCh := make(chan struct{}, 1)

	// Debounce reloads.
	go func() {
		var t *time.Timer
		for range resetCh {
			if t != nil {
				t.Stop()
			}
			t = time.AfterFunc(300*time.Millisecond, func() {
				newCfg, err := config.Load(configFile)
				if err != nil {
					logg.Error("config reload failed", "error", err)
					return
				}

				// Hot‑reload both watchers.
				fw.UpdateConfig(newCfg.WatchFS)
				sw.UpdateConfig(newCfg.Source)
				wkr.UpdateConfig(newCfg.Destination)

				logg.Info("config reloaded")
			})
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case ev := <-w.Events:
			if filepath.Base(ev.Name) != base {
				continue
			}
			if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
				continue
			}
			select {
			case resetCh <- struct{}{}:
			default:
			}

		case err := <-w.Errors:
			logg.Error("config watcher error", "error", err)
		}
	}
}
