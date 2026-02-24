// rdb-archiver is the entry point for the snapshot archiving service.
// It wires together the watcher, queue, worker, and filesystem layer.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/fs"
	"github.com/raoulx24/rdb-archiver/internal/watcher"
	"github.com/raoulx24/rdb-archiver/internal/worker"
)

func main() {
	// Create cancellable root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM for graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		cancel()
	}()

	// Configure paths
	rdbDir := "./data"        // directory where Redis/Valkey writes RDB files
	archiveDir := "./archive" // directory where snapshots are stored

	// Initialize components
	filesystem := fs.New()
	queue := worker.NewQueue(32)
	w := worker.New(filesystem, archiveDir)
	watch := watcher.New(rdbDir, queue)

	// Start worker loop
	go worker.RunLoop(ctx, w, queue)

	// Start watcher loop (poll every 2 seconds)
	go watch.Start(ctx, 2*time.Second)

	// Block until context is canceled
	<-ctx.Done()
	log.Println("exit complete")
}
