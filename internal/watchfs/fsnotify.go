package watchfs

import (
	"context"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// WatchFsNotify watches a directory and emits events when the target file changes.
func (w *FileWatcher) WatchFsNotify(
	ctx context.Context,
	dir, file string,
	events chan<- struct{},
) error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer fw.Close()

	if err := fw.Add(dir); err != nil {
		return err
	}

	resetCh := make(chan struct{}, 1)
	go w.debounceLoop(resetCh, events)

	for {
		select {
		case <-ctx.Done():
			return nil

		case ev := <-fw.Events:
			if filepath.Base(ev.Name) != file {
				continue
			}
			if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
				continue
			}
			select {
			case resetCh <- struct{}{}:
			default:
			}

		case <-fw.Errors:
			// errors are ignored; caller may log
		}
	}
}
