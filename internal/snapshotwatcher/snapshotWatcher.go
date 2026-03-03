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
func (sw *SnapshotWatcher) Start(ctx context.Context) error {
	sw.log.Info("starting snapshot watcher")
	go sw.consumeEvents()

	sw.mu.RLock()
	dir := sw.cfg.Path
	file := sw.cfg.PrimaryName
	mode := sw.cfg.WatchMode
	sw.mu.RUnlock()

	sw.checkForNewSnapshot()

	return sw.fileWatch.StartWatchingForFile(ctx, mode, dir, file, sw.events, sw.log)
}

// consumeEvents runs checkForNewSnapshot() for each incoming signal.
func (sw *SnapshotWatcher) consumeEvents() {
	for range sw.events {
		sw.checkForNewSnapshot()
	}
}

// CurrentConfig returns a copy of the current config.
func (sw *SnapshotWatcher) CurrentConfig() Config {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	return sw.cfg
}
