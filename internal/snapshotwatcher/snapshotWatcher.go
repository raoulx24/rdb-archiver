// Package snapshotwatcher detects new Redis snapshots and emits jobs.
package snapshotwatcher

import (
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/mailbox"
	"github.com/raoulx24/rdb-archiver/internal/watchfs"
)

type SnapshotWatcher struct {
	mu          sync.RWMutex
	dir         string
	primaryName string
	auxNames    []string
	lastModTime time.Time

	events      chan struct{}
	fileWatch   *watchfs.FileWatcher
	mb          *mailbox.Mailbox[Job]
	log         logging.Logger
	useFsNotify bool
}

// New creates a snapshot watcher with initial config.
func New(
	cfg config.SourceConfig,
	fw *watchfs.FileWatcher,
	mb *mailbox.Mailbox[Job],
	log logging.Logger,
	useFsNotify bool,
) *SnapshotWatcher {
	return &SnapshotWatcher{
		dir:         cfg.Path,
		primaryName: cfg.PrimaryName,
		auxNames:    cfg.AuxNames,
		fileWatch:   fw,
		mb:          mb,
		log:         log,
		useFsNotify: useFsNotify,
		events:      make(chan struct{}), // unbuffered
	}
}
