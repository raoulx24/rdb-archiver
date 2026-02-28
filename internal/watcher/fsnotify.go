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

	// Channel to request debounce resets
	resetCh := make(chan struct{}, 1)

	// Debounce goroutine
	go func() {
		var t *time.Timer
		for range resetCh {
			if t != nil {
				t.Stop()
			}
			t = time.AfterFunc(debounce, func() {
				defer func() {
					if r := recover(); r != nil {
						w.log.Error("detect panic", "panic", r)
					}
				}()
				w.detect()
			})
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case ev, ok := <-watcher.Events:
			if !ok {
				w.log.Error("events channel closed")
				return nil
			}

			w.log.Debug("event", "name", ev.Name, "op", ev.Op)

			if filepath.Base(ev.Name) != primary {
				continue
			}

			// Non-blocking send to reset debounce
			select {
			case resetCh <- struct{}{}:
			default:
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			w.log.Error("fsnotify error", "error", err)
		}
	}
}
