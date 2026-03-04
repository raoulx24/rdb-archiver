package main

import (
	"context"

	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/snapshotwatcher"
)

type SnapshotWatcherManager struct {
	sw     *snapshotwatcher.Watcher
	logg   logging.Logger
	cancel context.CancelFunc
}

func NewSnapshotWatcherManager(
	sw *snapshotwatcher.Watcher,
	logg logging.Logger,
) *SnapshotWatcherManager {
	return &SnapshotWatcherManager{
		sw:   sw,
		logg: logg,
	}
}

func (m *SnapshotWatcherManager) Start(ctx context.Context) {
	if m.cancel != nil {
		m.cancel()
	}

	var wctx context.Context
	wctx, m.cancel = context.WithCancel(ctx)

	go func() {
		if err := m.sw.Start(wctx); err != nil {
			m.logg.Error("snapshot watcher stopped", "error", err)
		}
	}()
}

func (m *SnapshotWatcherManager) Stop() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}
