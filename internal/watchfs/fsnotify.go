package watchfs

import (
	"context"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// WatchFsNotify watches a directory and emits events when the target file changes.
func (wfs *FileWatcher) WatchFsNotify(
	ctx context.Context,
	dir, file string,
	events chan<- struct{},
) error {
	wfs.logg.Info("starting watch fs - fsnotify mode")

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		wfs.logg.Error("failed to create fsnotify watcher", "error", err)
		return err
	}
	defer fw.Close()

	if err := fw.Add(dir); err != nil {
		return err
	}

	resetCh := make(chan struct{}, 1)

	go wfs.debounceLoop(ctx, dir, file, resetCh, events)

	for {
		select {
		case <-ctx.Done():
			wfs.logg.Debug("stopping fsnotify watcher", "function", "WatchFsNotify")
			return nil

		case ev := <-fw.Events:
			wfs.logg.Debug("event received", "function", "WatchFsNotify", "fsnotifyEvent", ev)
			if filepath.Base(ev.Name) != file {
				continue
			}
			if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
				continue
			}

			wfs.logg.Debug("event received", "function", "WatchFsNotify", "fsnotifyEvent", ev.Op)

			// Non-blocking send to collapse bursts
			select {
			case resetCh <- struct{}{}:
			default:
			}

		case <-fw.Errors:
			wfs.logg.Warn("fsnotify error", "error", err)
		}
	}
}
