package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/retention"
	"github.com/raoulx24/rdb-archiver/internal/watcher"
	"github.com/raoulx24/rdb-archiver/internal/worker"
)

func main() {
	// Root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		cancel()
	}()

	// Load config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize components
	logg := logging.StdLogger{}
	ret := retention.New(&cfg, logg)

	queue := worker.NewQueue(32)

	// Worker constructor: (archiveDir, logger, retentionEngine)
	w := worker.New(
		cfg.Destination.Root,
		logg,
		ret,
	)

	// Watcher constructor: (rdbDir, queue)
	watch := watcher.New(
		cfg.Source.Path,
		queue,
	)

	// Start worker loop
	go worker.RunLoop(ctx, w, queue)

	// Start watcher loop (poll every 2 seconds)
	go watch.Start(ctx, 2*time.Second)

	// Block until shutdown
	<-ctx.Done()
	log.Println("exit complete")
}
