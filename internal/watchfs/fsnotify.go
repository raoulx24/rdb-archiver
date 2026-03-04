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
		return err
	}
	defer fw.Close()

	if err := fw.Add(dir); err != nil {
		return err
	}

	resetCh := make(chan struct{}, 1)

	// NEW: pass ctx into debounceLoop
	go wfs.debounceLoop(ctx, resetCh, events)

	for {
		select {
		case <-ctx.Done():
			// debounceLoop will exit automatically because it also listens to ctx.Done()
			return nil

		case ev := <-fw.Events:
			if filepath.Base(ev.Name) != file {
				continue
			}
			if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
				continue
			}

			wfs.logg.Debug("event received", "fsnotifyEvent", ev.Op)

			// Non-blocking send to collapse bursts
			select {
			case resetCh <- struct{}{}:
			default:
			}

		case <-fw.Errors:
			// ignore fsnotify errors; caller may log
		}
	}
}
