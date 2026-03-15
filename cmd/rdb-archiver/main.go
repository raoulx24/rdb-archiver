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

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configFile := "config/config.yaml"
	stdLog := log.New(os.Stdout, "", log.LstdFlags)

	cfg, fileHash, err := config.Load(configFile)
	if err != nil {
		stdLog.Fatalf("failed to load config: %v", err)
	}
	cfg.ApplyDefaults(stdLog)

	logg := logging.NewSlogLogger(cfg.Logging)
	mainLogg := logg.With("pkg", "main")

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		mainLogg.Info("shutting down...")
		cancel()
	}()

	// metrics server - prometheus
	mux, metricsBundle := buildMetrics(cfg.Prometheus)
	metricsSrv, metricsCancel := startPrometheus(ctx, cfg.Prometheus, logg, mux)
	// metrics server - end of

	osfs := fs.New(cfg.FS, logg)
	mb := mailbox.New[snapshot.Job](metricsBundle.Mailbox, logg)
	ret := retention.New(logg, metricsBundle.Retention)

	fw, err := watchfs.New(cfg.WatchFS, logg)
	if err != nil {
		logg.Error("invalid params for watchfs", "error", err)
		os.Exit(1)
	}

	mainWorker := worker.New(cfg.Destination, logg, metricsBundle.Worker, ret, mb, osfs)
	go mainWorker.Start(ctx)

	snapWatcher := snapshotwatcher.New(cfg.Source, metricsBundle.SnapshotWatcher, fw, mb, logg)
	swm := NewSnapshotWatcherManager(snapWatcher, logg)
	swm.Start(ctx)

	if cfg.ConfigReload.Enabled {
		reloader := NewConfigReloader(
			configFile,
			fileHash,
			cfg.ConfigReload.Method,
			fw,
			mainLogg,
			func(newCfg *config.Config) {
				logg.UpdateConfig(newCfg.Logging)
				_ = fw.UpdateConfig(newCfg.WatchFS)
				osfs.UpdateConfig(newCfg.FS)
				mainWorker.UpdateConfig(newCfg.Destination)

				if snapWatcher.NeedsRestart(newCfg.Source) {
					snapWatcher.UpdateConfig(newCfg.Source)
					swm.Start(ctx)
				}

				// restart prometheus if needed
				if metricsSrv.NeedsRestart(newCfg.Prometheus) {
					mainLogg.Info("restarting prometheus server")

					_ = metricsSrv.Stop(context.Background())
					metricsCancel()

					if metricsSrv.NeedMetricsRebuild(newCfg.Prometheus) {
						newMux, metricsBundle := buildMetrics(cfg.Prometheus)
						mux = newMux

						mb.RebuildMetrics(metricsBundle.Mailbox)
						ret.RebuildMetrics(metricsBundle.Retention)
						mainWorker.RebuildMetrics(metricsBundle.Worker)
						snapWatcher.RebuildMetrics(metricsBundle.SnapshotWatcher)
					}

					metricsSrv, metricsCancel = startPrometheus(ctx, newCfg.Prometheus, logg, mux)
				}

			},
		)
		go reloader.Start(ctx)
	}

	healthSrv := health.New(cfg.Health, logg, snapWatcher)
	go func() {
		if err := healthSrv.Start(ctx); err != nil {
			mainLogg.Error("health server stopped", "error", err)
		}
	}()

	<-ctx.Done()

	mb.Stop()

	stdLog.Println("exit complete")
}
