package watchfs

import (
	"time"
)

// debounceLoop collapses bursts of events and enforces a stability delay.
func (wfs *FileWatcher) debounceLoop(
	resetCh <-chan struct{},
	events chan<- struct{},
) {
	var t *time.Timer

	for range resetCh {
		if t != nil {
			t.Stop()
		}

		wfs.mu.RLock()
		debounce := wfs.debounceWindow
		stability := wfs.stabilityWindow
		wfs.mu.RUnlock()

		t = time.AfterFunc(debounce, func() {
			wfs.logg.Debug("debounce loop triggered", "debounceWindow", debounce, "stabilityWindow", stability)
			time.Sleep(stability)
			events <- struct{}{}
		})
	}
}
