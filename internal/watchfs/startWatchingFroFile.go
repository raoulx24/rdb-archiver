package watchfs

import (
	"context"

	"github.com/raoulx24/rdb-archiver/internal/fsprobe"
)

// StartWatchingForFile chooses mode and starts watching a single file.
func (wfs *FileWatcher) StartWatchingForFile(
	ctx context.Context,
	mode string,
	dir string,
	file string,
	events chan<- struct{},
) error {
	wfs.logg.Debug("starting watch fs", "watchFSMode", mode)
	switch mode {
	case "fsnotify":
		return wfs.WatchFsNotify(ctx, dir, file, events)

	case "poll":
		return wfs.WatchPolling(ctx, events)

	case "auto":
		res := fsprobe.Probe(dir)
		if res.FsnotifySupported {
			wfs.logg.Debug("fsnotify supported", "dir", dir)
			return wfs.WatchFsNotify(ctx, dir, file, events)
		}
		wfs.logg.Error("fsnotify disabled, falling back to polling", "reason", res.Reason)
		return wfs.WatchPolling(ctx, events)

	default:
		wfs.logg.Error("invalid watch mode, using polling", "mode", mode)
		return wfs.WatchPolling(ctx, events)
	}
}
