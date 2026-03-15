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
	mu            sync.RWMutex
	cfg           Config
	metrics       Metrics
	lastModTime   time.Time
	events        chan struct{}
	fileWatch     *watchfs.FileWatcher
	mb            *mailbox.Mailbox[snapshot.Job]
	lastHeartbeat time.Time
	timerTick     time.Duration
	logg          logging.Logger
}

// New creates a snapshotwatcher watcher with initial config.
func New(
	cfg Config,
	metrics Metrics,
	fw *watchfs.FileWatcher,
	mb *mailbox.Mailbox[snapshot.Job],
	log logging.Logger,
) *Watcher {
	logg := log.With("pkg", "snapshotwatcher")
	logg.Debug("creating snapshot watcher", "function", "New")
	return &Watcher{
		cfg:           cfg,
		metrics:       metrics,
		fileWatch:     fw,
		mb:            mb,
		lastHeartbeat: time.Now(),
		timerTick:     20 * time.Second,
		logg:          logg,
		events:        make(chan struct{}), // unbuffered
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
	metrics := sw.metrics
	sw.mu.RUnlock()

	sw.checkForNewSnapshot(metrics)

	sw.mu.Lock()
	sw.lastHeartbeat = time.Now()
	sw.mu.Unlock()

	return sw.fileWatch.StartWatchingForFile(ctx, mode, dir, file, sw.events)
}

// consumeEvents runs checkForNewSnapshot() for each incoming signal.
func (sw *Watcher) consumeEvents(ctx context.Context, events <-chan struct{}) {
	ticker := time.NewTicker(sw.timerTick)
	defer ticker.Stop()
	sw.logg.Debug("consuming events", "function", "consumeEvents")

	for {
		select {
		case <-ctx.Done():
			sw.logg.Info("stopping snapshot watcher event loop")
			return
		case <-ticker.C:
			sw.mu.Lock()
			sw.lastHeartbeat = time.Now()
			sw.mu.Unlock()
			sw.logg.Debug("last heartbeat ticker", "function", "consumeEvents")
		case _, ok := <-events:
			if !ok {
				sw.logg.Info("events channel closed, stopping event loop")
				return
			}
			sw.mu.Lock()
			sw.lastHeartbeat = time.Now()
			metrics := sw.metrics
			sw.mu.Unlock()
			metrics.EventReceived()
			sw.logg.Debug("event received", "function", "consumeEvents")
			sw.checkForNewSnapshot(metrics)
		}
	}
}

func (sw *Watcher) IsAlive(maxSilence time.Duration) bool {
	sw.mu.RLock()
	elapsedTime := time.Since(sw.lastHeartbeat)
	sw.mu.RUnlock()
	sw.logg.Debug("queried for is alive", "function", "IsAlive", "elapsedTime", elapsedTime, "maxSilence", maxSilence)
	return elapsedTime < maxSilence
}
