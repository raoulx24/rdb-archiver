package watchfs

import (
	"context"
	"path/filepath"
	"time"
)

// debounceLoop collapses bursts of events and enforces a stability delay.
// Both debounce and stability windows are restartable on new resets.
func (wfs *FileWatcher) debounceLoop(
	ctx context.Context,
	dir string,
	file string,
	resetCh <-chan struct{},
	events chan<- struct{},
) {
	var (
		timer   *time.Timer
		timerCh <-chan time.Time
	)
	wfs.logg.Debug("debounce started", "function", "debounceLoop")

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			wfs.logg.Debug("debounce stopping", "function", "debounceLoop")
			return

		case <-resetCh:
		restart:
			wfs.logg.Debug("debounce loop starts", "function", "debounceLoop")
			// Stop old timer if needed.
			if timer != nil {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
			}

			// Load current config.
			wfs.mu.RLock()
			debounce := wfs.debounceWindow
			stability := wfs.stabilityWindow
			wfs.mu.RUnlock()

			// Start debounce timer.
			timer = time.NewTimer(debounce)
			timerCh = timer.C

			// Wait for debounce to fire or for more resets.
			select {
			case <-ctx.Done():
				timer.Stop()
				wfs.logg.Debug("debounce loop - debounce stopping", "function", "debounceLoop")
				return

			case <-resetCh:
				// Restart debounce.
				wfs.logg.Debug("debounce loop restart", "function", "debounceLoop")
				goto restart

			case <-timerCh:
				wfs.logg.Debug("debounce loop finished", "function", "debounceLoop")
				// Debounce window completed; now stability phase.
			}

			wfs.logg.Debug("stability phase started", "function", "debounceLoop")
			// Stability phase: must remain quiet for stability duration.
			if stability > 0 {
				stabTimer := time.NewTimer(stability)

				select {
				case <-ctx.Done():
					stabTimer.Stop()
					wfs.logg.Debug("stability phase - debounce stopping", "function", "debounceLoop")
					return

				case <-resetCh:
					stabTimer.Stop()
					wfs.logg.Debug("stability phase - debounce loop restart", "function", "debounceLoop")
					goto restart

				case <-stabTimer.C:
					wfs.logg.Debug("stability phase finished", "function", "debounceLoop")
					// Stability achieved.
				}
			}

			if !wfs.isWatchedFileChanged(filepath.Join(dir, file)) {
				wfs.logg.Debug("the file is not changed", "function", "debounceLoop")
				return
			}

			// Emit event unless shutting down.
			select {
			case <-ctx.Done():
				wfs.logg.Debug("after stability window - debounce stopping", "function", "debounceLoop")
				return
			case events <- struct{}{}:
				wfs.logg.Debug("file changed event emitted", "function", "debounceLoop")
			}

			// Reset state.
			timer = nil
			timerCh = nil
		}
	}
}
