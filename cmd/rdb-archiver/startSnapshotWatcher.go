package main

import (
	"context"

	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/snapshotwatcher"
)

func startSnapshotWatcher(ctx context.Context, sw *snapshotwatcher.SnapshotWatcher, logg logging.Logger) {
	if snapshotCancel != nil {
		snapshotCancel()
	}

	var wctx context.Context
	wctx, snapshotCancel = context.WithCancel(ctx)

	go func() {
		if err := sw.Start(wctx); err != nil {
			logg.Error("snapshot watcher stopped", "error", err)
		}
	}()
}
