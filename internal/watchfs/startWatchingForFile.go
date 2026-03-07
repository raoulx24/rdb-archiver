package watchfs

import (
	"context"
	"os"
	"path/filepath"

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

	// detect symlink
	info, err := os.Lstat(filepath.Join(dir, file))
	if err == nil && (info.Mode()&os.ModeSymlink != 0) {
		if mode != "poll" {
			wfs.logg.Warn("file is a symlink, using polling mode by default", "dir", dir, "file", file)
		}
		return wfs.WatchPolling(ctx, dir, file, events)
	}

	switch mode {
	case "fsnotify":
		return wfs.WatchFsNotify(ctx, dir, file, events)

	case "poll":
		return wfs.WatchPolling(ctx, dir, file, events)

	case "auto":
		res := fsprobe.Probe(dir)
		if res.FsnotifySupported {
			wfs.logg.Debug("fsnotify supported", "dir", dir)
			return wfs.WatchFsNotify(ctx, dir, file, events)
		}
		wfs.logg.Error("fsnotify disabled, falling back to polling", "reason", res.Reason)
		return wfs.WatchPolling(ctx, dir, file, events)

	default:
		wfs.logg.Error("invalid watch mode, using polling", "mode", mode)
		return wfs.WatchPolling(ctx, dir, file, events)
	}
}
