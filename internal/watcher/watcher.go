// Package watcher monitors the RDB directory for new or updated snapshot files
// and submits jobs to the worker queue.
package watcher

import (
	"context"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/worker"
)

// Watcher observes a directory for RDB file changes and pushes jobs into a queue.
type Watcher struct {
	dir   string
	queue *worker.Queue
}

func New(dir string, q *worker.Queue) *Watcher {
	return &Watcher{
		dir:   dir,
		queue: q,
	}
}

// Start begins the polling loop that checks for new or modified RDB files.
func (w *Watcher) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	seen := make(map[string]time.Time)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scan(ctx, seen)
		}
	}
}
