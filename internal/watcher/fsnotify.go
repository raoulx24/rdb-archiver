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
	primary := w.primaryName
	w.mu.RUnlock()

	if err := watcher.Add(dir); err != nil {
		return err
	}

	var (
		timer  *time.Timer
		timerC <-chan time.Time
	)

	for {
		select {
		case <-ctx.Done():
			return nil

		case ev := <-watcher.Events:
			if filepath.Base(ev.Name) != primary {
				continue
			}

			// Start or reset debounce timer
			if timer != nil {
				if !timer.Stop() {
					<-timer.C // drain if needed
				}
			}

			timer = time.NewTimer(debounce)
			timerC = timer.C

		case <-timerC:
			// Debounce window passed without new events
			w.detect()
			timerC = nil

		case <-watcher.Errors:
			// ignore errors
		}
	}
}
