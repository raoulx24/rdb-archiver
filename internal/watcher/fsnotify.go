package watcher

import (
	"context"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// StartFsNotify triggers detect() when fsnotify reports relevant changes.
func (w *Watcher) StartFsNotify(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	w.mu.RLock()
	dir := w.dir
	debounce := w.debounce
	w.mu.RUnlock()

	if err := watcher.Add(dir); err != nil {
		return err
	}

	var last time.Time

	for {
		select {
		case <-ctx.Done():
			return nil

		case ev := <-watcher.Events:
			w.mu.RLock()
			primary := w.primaryName
			w.mu.RUnlock()

			if filepath.Base(ev.Name) != primary {
				continue
			}

			if time.Since(last) < debounce {
				continue
			}
			last = time.Now()

			w.detect()

		case <-watcher.Errors:
			// ignore errors
		}
	}
}
