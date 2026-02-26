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
	"github.com/raoulx24/rdb-archiver/internal/watcher"
	"github.com/raoulx24/rdb-archiver/internal/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Temporary fallback logger
	stdLog := log.New(os.Stdout, "", log.LstdFlags)

	// 1️⃣ Load config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		stdLog.Fatalf("failed to load config: %v", err)
	}

	// Logger
	logg := logging.NewSlogLogger(cfg.Logging.Level, cfg.Logging.Format)

	// Graceful shutdown
	go func(logg logging.Logger) {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logg.Info("shutting down...")
		cancel()
	}(logg)

	// Mailbox for snapshot jobs
	mb := mailbox.New[worker.Job]()

	// Retention engine (promotion + cleanup)
	ret := retention.New(cfg, logg)

	// Worker (snapshot writer + promotion + cleanup)
	w := worker.New(
		cfg.Destination,
		logg,
		ret,
		mb,
		nil, // fs.FS (nil = default)
	)

	// Watcher (detects snapshots and pushes into mailbox)
	watch := watcher.New(
		cfg.Source,
		logg,
		mb,
	)

	// Start worker loop
	go w.Start(ctx)

	// Start watcher loop
	go func() {
		err := watch.Start(ctx)
		if err != nil {
			logg.Error("failed to start watcher", "error", err)
			os.Exit(1)
		}
	}()

	// Hot reload on SIGHUP
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP)

		for range sigCh {
			newCfg, err := config.Load("config.yaml")
			if err != nil {
				logg.Error("config reload failed", "error", err)
				continue
			}

			// Apply updates
			w.UpdateConfig(newCfg.Destination)
			watch.UpdateConfig(newCfg.Source)
			ret.UpdateConfig(newCfg)

			logg.Info("config reloaded")
		}
	}()

	<-ctx.Done()
	log.Println("exit complete")
}
