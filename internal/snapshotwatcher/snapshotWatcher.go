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

type SnapshotWatcher struct {
	mu          sync.RWMutex
	cfg         Config
	lastModTime time.Time
	events      chan struct{}
	fileWatch   *watchfs.FileWatcher
	mb          *mailbox.Mailbox[snapshot.Job]
	log         logging.Logger
}

// New creates a snapshotwatcher watcher with initial config.
func New(
	cfg Config,
	fw *watchfs.FileWatcher,
	mb *mailbox.Mailbox[snapshot.Job],
	log logging.Logger,
) *SnapshotWatcher {
	return &SnapshotWatcher{
		cfg:       cfg,
		fileWatch: fw,
		mb:        mb,
		log:       log,
		events:    make(chan struct{}), // unbuffered
	}
}

// Start begins watching using fsnotify or polling.
func (w *SnapshotWatcher) Start(ctx context.Context) error {
	go w.consumeEvents()

	w.mu.RLock()
	dir := w.cfg.Path
	file := w.cfg.PrimaryName
	mode := w.cfg.WatchMode
	w.mu.RUnlock()

	return w.fileWatch.StartWatchingForFile(ctx, mode, dir, file, w.events, w.log)
}

// consumeEvents runs detect() for each incoming signal.
func (w *SnapshotWatcher) consumeEvents() {
	for range w.events {
		w.detect()
	}
}
