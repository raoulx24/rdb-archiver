package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/fs"
	"github.com/raoulx24/rdb-archiver/internal/health"
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
	stdLog := log.New(os.Stdout, "", log.LstdFlags)

	cfg, err := config.Load(configFile)
	if err != nil {
		stdLog.Fatalf("failed to load config: %v", err)
	}
	cfg.ApplyDefaults()

	logg := logging.NewSlogLogger(cfg.Logging)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logg.Info("shutting down...")
		cancel()
	}()

	osfs := fs.New(cfg.FS)
	mb := mailbox.New[snapshot.Job]()
	ret := retention.New(logg)

	fw, err := watchfs.New(cfg.WatchFS, logg)
	if err != nil {
		logg.Error("invalid params for watchfs", "error", err)
		os.Exit(1)
	}

	mainWorker := worker.New(cfg.Destination, logg, ret, mb, osfs)
	go mainWorker.Start(ctx)

	snapWatcher := snapshotwatcher.New(cfg.Source, fw, mb, logg)
	swm := NewSnapshotWatcherManager(snapWatcher, logg)
	swm.Start(ctx)

	if cfg.ConfigReload.Enabled {
		reloader := NewConfigReloader(
			configFile,
			cfg.ConfigReload.Method,
			fw,
			logg,
			func(newCfg *config.Config) {
				logg.UpdateConfig(newCfg.Logging)
				fw.UpdateConfig(newCfg.WatchFS)
				osfs.UpdateConfig(newCfg.FS)
				mainWorker.UpdateConfig(newCfg.Destination)

				oldSnapCfg := snapWatcher.CurrentConfig()
				snapWatcher.UpdateConfig(newCfg.Source)
				if snapWatcher.NeedsRestart(oldSnapCfg, newCfg.Source) {
					swm.Start(ctx)
				}
			},
		)
		go reloader.Start(ctx)
	}

	healthSrv := health.New(cfg.Health, snapWatcher)
	go func() {
		if err := healthSrv.Start(ctx); err != nil {
			logg.Error("health server stopped", "error", err)
		}
	}()

	<-ctx.Done()
	stdLog.Println("exit complete")
}
