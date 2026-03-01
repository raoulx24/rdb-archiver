package watchfs

import (
	"time"
)

// debounceLoop collapses bursts of events and enforces a stability delay.
func (w *FileWatcher) debounceLoop(
	resetCh <-chan struct{},
	events chan<- struct{},
) {
	var t *time.Timer

	for range resetCh {
		if t != nil {
			t.Stop()
		}

		w.mu.RLock()
		debounce := w.debounceWindow
		stability := w.stabilityWindow
		w.mu.RUnlock()

		t = time.AfterFunc(debounce, func() {
			time.Sleep(stability)
			events <- struct{}{}
		})
	}
}
