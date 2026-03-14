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
	"github.com/raoulx24/rdb-archiver/internal/metrics/prometheus"
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
	cfg.ApplyDefaults()

	logg := logging.NewSlogLogger(cfg.Logging)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logg.Info("shutting down...")
		cancel()
	}()

	// metrics server - prometheus
	metricsReg := prometheus.NewRegistry()
	workerMetrics := prometheus.NewWorkerMetrics(metricsReg)
	snapshotWatcherMetrics := prometheus.NewSnapshotWatcherMetrics(metricsReg)
	mailboxMetrics := prometheus.NewMailboxMetrics(metricsReg)
	retentionMetrics := prometheus.NewRetentionMetrics(metricsReg)

	metricsHandler := metricsReg.Handler()
	metricsSrv := prometheus.New(cfg.Prometheus, metricsHandler)

	go func() {
		logg.Info("starting metrics server", "addr", ":9090")
		if err := metricsSrv.Start(ctx); err != nil {
			logg.Error("metrics server stopped", "error", err)
		}
	}()
	// metrics server - end of

	osfs := fs.New(cfg.FS, logg)
	mb := mailbox.New[snapshot.Job](mailboxMetrics)
	ret := retention.New(logg, retentionMetrics)

	fw, err := watchfs.New(cfg.WatchFS, logg)
	if err != nil {
		logg.Error("invalid params for watchfs", "error", err)
		os.Exit(1)
	}

	mainWorker := worker.New(cfg.Destination, logg, workerMetrics, ret, mb, osfs)
	go mainWorker.Start(ctx)

	snapWatcher := snapshotwatcher.New(cfg.Source, snapshotWatcherMetrics, fw, mb, logg)
	swm := NewSnapshotWatcherManager(snapWatcher, logg)
	swm.Start(ctx)

	if cfg.ConfigReload.Enabled {
		reloader := NewConfigReloader(
			configFile,
			fileHash,
			cfg.ConfigReload.Method,
			fw,
			logg,
			func(newCfg *config.Config) {
				logg.UpdateConfig(newCfg.Logging)
				_ = fw.UpdateConfig(newCfg.WatchFS)
				osfs.UpdateConfig(newCfg.FS)
				mainWorker.UpdateConfig(newCfg.Destination)

				if snapWatcher.NeedsRestart(newCfg.Source) {
					snapWatcher.UpdateConfig(newCfg.Source)
					swm.Start(ctx)
				}
			},
		)
		go reloader.Start(ctx)
	}

	healthSrv := health.New(cfg.Health, snapWatcher)
	go func() {
		logg.Info("starting health server")
		if err := healthSrv.Start(ctx); err != nil {
			logg.Error("health server stopped", "error", err)
		}
	}()

	<-ctx.Done()

	mb.Stop()

	stdLog.Println("exit complete")
}
