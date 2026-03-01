package watchfs

import (
	"context"

	"github.com/raoulx24/rdb-archiver/internal/fsprobe"
	"github.com/raoulx24/rdb-archiver/internal/logging"
)

func (w *FileWatcher) StartWatchingForFile(
	ctx context.Context,
	mode string,
	dir string,
	file string,
	events chan<- struct{},
	log logging.Logger,
) error {
	switch mode {
	case "fsnotify":
		return w.WatchFsNotify(ctx, dir, file, events)

	case "poll":
		return w.WatchPolling(ctx, events)

	case "auto":
		res := fsprobe.Probe(dir)
		if res.FsnotifySupported {
			log.Debug("fsnotify supported", "dir", dir)
			return w.WatchFsNotify(ctx, dir, file, events)
		}
		log.Error("fsnotify disabled, falling back to polling", "reason", res.Reason)
		return w.WatchPolling(ctx, events)

	default:
		log.Error("invalid watch mode, using polling", "mode", mode)
		return w.WatchPolling(ctx, events)
	}
}
