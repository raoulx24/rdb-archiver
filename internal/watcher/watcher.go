// Package watcher monitors the source directory and emits snapshot jobs.
package watcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/fsprobe"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/mailbox"
	"github.com/raoulx24/rdb-archiver/internal/worker"
)

// Watcher observes the primary file and enqueues new snapshots when updated.
type Watcher struct {
	mu sync.RWMutex

	dir         string
	primaryName string
	auxNames    []string
	interval    time.Duration
	mode        string
	debounce    time.Duration
	stability   time.Duration

	log logging.Logger

	lastModTime time.Time

	mb *mailbox.Mailbox[worker.Job]
}

// New creates a watcher from the source configuration.
func New(cfg config.SourceConfig, log logging.Logger, mb *mailbox.Mailbox[worker.Job]) *Watcher {
	return &Watcher{
		dir:         cfg.Path,
		primaryName: cfg.PrimaryName,
		auxNames:    cfg.AuxNames,
		interval:    cfg.Watch.PollInterval,
		mode:        cfg.Watch.Mode,
		debounce:    cfg.Watch.DebounceWindow,
		stability:   cfg.Watch.StabilityWindow,
		log:         log,
		mb:          mb,
	}
}

// Start chooses the correct watching strategy based on config.
func (w *Watcher) Start(ctx context.Context) error {
	switch w.mode {
	case "fsnotify":
		return w.StartFsNotify(ctx)

	case "poll":
		w.StartPolling(ctx)
		return nil

	case "auto":
		res := fsprobe.Probe(w.dir)
		if res.FsnotifySupported {
			return w.StartFsNotify(ctx)
		}
		w.log.Warn("fsnotify disabled: %s", res.Reason)
		w.StartPolling(ctx)
		return nil

	default:
		return fmt.Errorf("unknown mode %q", w.mode)
	}
}
