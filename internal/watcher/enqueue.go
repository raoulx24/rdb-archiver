package watcher

import (
	"context"
	"log"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/worker"
)

// enqueue.go submits a new snapshot job to the worker queue.
func (w *Watcher) enqueue(ctx context.Context, path string, mod time.Time) {
	select {
	case <-ctx.Done():
		log.Printf("watcher: context canceled before enqueue")
		return
	default:
		// safe because queue is buffered
		w.queue.Push(worker.Job{
			SourcePath: path,
			Timestamp:  mod,
		})
		log.Printf("watcher: queued snapshot %s", path)
	}
}
