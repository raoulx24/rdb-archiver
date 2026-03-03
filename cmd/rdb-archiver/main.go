package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/fs"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/mailbox"
	"github.com/raoulx24/rdb-archiver/internal/retention"
	"github.com/raoulx24/rdb-archiver/internal/snapshot"
	"github.com/raoulx24/rdb-archiver/internal/snapshotwatcher"
	"github.com/raoulx24/rdb-archiver/internal/watchfs"
	"github.com/raoulx24/rdb-archiver/internal/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configFile := "config/config.yaml"

	// Temporary fallback logger
	stdLog := log.New(os.Stdout, "", log.LstdFlags)

	// Load config
	cfg, err := config.Load(configFile)
	if err != nil {
		stdLog.Fatalf("failed to load config: %v", err)
	}

	// Logger
	logg := logging.NewSlogLogger(cfg.Logging)

	// Graceful shutdown
	go func(logg logging.Logger) {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logg.Info("shutting down...")
		cancel()
	}(logg)

	// OSFS
	osfs := fs.New(cfg.FS)

	// Mailbox for snapshotwatcher jobs
	mb := mailbox.New[snapshot.Job]()

	// Retention engine
	ret := retention.New(logg)

	// FileWatcher
	fw, err := watchfs.New(cfg.WatchFS)
	if err != nil {
		panic(err)
	}

	// Worker
	mainWorker := worker.New(
		cfg.Destination,
		logg,
		ret,
		mb,
		osfs,
	)

	// Snapshot watcher
	snapWatcher := snapshotwatcher.New(
		cfg.Source,
		fw,
		mb,
		logg,
	)

	// Start worker loop
	go mainWorker.Start(ctx)

	// Start snapshot watcher (restartable)
	startSnapshotWatcher(ctx, snapWatcher, logg)

	// Config reload
	if cfg.ConfigReload.Enabled {
		go startConfigReload(ctx, fw, snapWatcher, mainWorker, osfs, logg, configFile, cfg.ConfigReload.Method)
	}

	<-ctx.Done()
	stdLog.Println("exit complete")
}

// helper functions
var snapshotCancel context.CancelFunc

func startSnapshotWatcher(ctx context.Context, sw *snapshotwatcher.SnapshotWatcher, logg logging.Logger) {
	if snapshotCancel != nil {
		snapshotCancel()
	}

	var wctx context.Context
	wctx, snapshotCancel = context.WithCancel(ctx)

	go func() {
		if err := sw.Start(wctx); err != nil {
			logg.Error("snapshot watcher stopped", "error", err)
		}
	}()
}

func startConfigReload(
	ctx context.Context,
	fw *watchfs.FileWatcher,
	sw *snapshotwatcher.SnapshotWatcher,
	wkr *worker.Worker,
	osfs *fs.OSFS,
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

				// filesystem
				osfs.UpdateConfig(newCfg.FS)

				logg.Info("config reloaded")
			})
		}
	}
}
