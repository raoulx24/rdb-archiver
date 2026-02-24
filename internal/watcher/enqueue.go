package watcher

import (
	"context"
	"log"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/worker"
)

// submits a new snapshot job to the worker queue.

func (w *Watcher) enqueue(ctx context.Context, path string, mod time.Time) {
	job := worker.Job{
		SourcePath: path,
		Timestamp:  mod,
	}

	select {
	case w.queue.Ch <- job:
		log.Printf("watcher: queued snapshot %s", path)
	case <-ctx.Done():
		log.Printf("watcher: context canceled before enqueue")
	}
}
