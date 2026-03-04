// Package snapshotwatcher detects new Redis snapshots and emits jobs.
package snapshotwatcher

import (
	"context"
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/mailbox"
	"github.com/raoulx24/rdb-archiver/internal/snapshot"
	"github.com/raoulx24/rdb-archiver/internal/watchfs"
)

type Watcher struct {
	mu          sync.RWMutex
	cfg         Config
	lastModTime time.Time
	events      chan struct{}
	fileWatch   *watchfs.FileWatcher
	mb          *mailbox.Mailbox[snapshot.Job]
	logg        logging.Logger
}

// New creates a snapshotwatcher watcher with initial config.
func New(
	cfg Config,
	fw *watchfs.FileWatcher,
	mb *mailbox.Mailbox[snapshot.Job],
	log logging.Logger,
) *Watcher {
	logg := log.With("pkg", "snapshotwatcher")
	logg.Debug("creating snapshot watcher")
	return &Watcher{
		cfg:       cfg,
		fileWatch: fw,
		mb:        mb,
		logg:      logg,
		events:    make(chan struct{}), // unbuffered
	}
}

// Start begins watching using fsnotify or polling.
func (sw *Watcher) Start(ctx context.Context) error {
	sw.logg.Info("starting snapshot watcher")
	// Create a fresh event channel per start.
	sw.mu.Lock()
	sw.events = make(chan struct{})
	events := sw.events
	sw.mu.Unlock()
	go sw.consumeEvents(ctx, events)

	sw.mu.RLock()
	dir := sw.cfg.Path
	file := sw.cfg.PrimaryName
	mode := sw.cfg.WatchMode
	sw.mu.RUnlock()

	sw.checkForNewSnapshot()

	return sw.fileWatch.StartWatchingForFile(ctx, mode, dir, file, sw.events)
}

// consumeEvents runs checkForNewSnapshot() for each incoming signal.
func (sw *Watcher) consumeEvents(ctx context.Context, events <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			sw.logg.Info("stopping snapshot watcher event loop")
			return
		case _, ok := <-events:
			if !ok {
				sw.logg.Info("events channel closed, stopping event loop")
				return
			}
			sw.checkForNewSnapshot()
		}
	}
}

// CurrentConfig returns a copy of the current config.
func (sw *Watcher) CurrentConfig() Config {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	return sw.cfg
}
