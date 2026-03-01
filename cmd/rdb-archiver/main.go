package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/raoulx24/rdb-archiver/internal/config"
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

	// 1️⃣ Load config
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

	// Mailbox for snapshotwatcher jobs
	mb := mailbox.New[snapshot.Job]()

	// Retention engine (promotion + cleanup)
	ret := retention.New(logg)

	// FileWatcher handles fsnotify/polling + debounce.
	fw, err := watchfs.New(cfg.WatchFS)
	if err != nil {
		panic(err)
	}

	// Worker (snapshotwatcher writer + promotion + cleanup)
	mainWorker := worker.New(
		cfg.Destination,
		logg,
		ret,
		mb,
		nil, // fs.FS (nil = default)
	)

	// Watcher (detects snapshots and pushes into mailbox)
	snapWatcher := snapshotwatcher.New(
		cfg.Source,
		fw,
		mb,
		logg,
	)

	// Start worker loop
	go mainWorker.Start(ctx)

	// Start watcher loop
	go func() {
		err := snapWatcher.Start(ctx)
		if err != nil {
			logg.Error("failed to start snapshotwatcher watcher", "error", err)
			os.Exit(1)
		}
	}()

	// Start config hot‑reload watcher
	go startConfigReload(ctx, fw, snapWatcher, mainWorker, logg)

	<-ctx.Done()
	stdLog.Println("exit complete")
}
