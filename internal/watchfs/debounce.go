package watchfs

import (
	"context"
	"time"
)

// debounceLoop collapses bursts of events and enforces a stability delay.
// Both debounce and stability windows are restartable on new resets.
func (wfs *FileWatcher) debounceLoop(
	ctx context.Context,
	resetCh <-chan struct{},
	events chan<- struct{},
) {
	var (
		timer   *time.Timer
		timerCh <-chan time.Time
	)

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return

		case <-resetCh:
		restart:
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
				return

			case <-resetCh:
				// Restart debounce.
				goto restart

			case <-timerCh:
				// Debounce window completed; now stability phase.
			}

			// Stability phase: must remain quiet for stability duration.
			if stability > 0 {
				stabTimer := time.NewTimer(stability)

				select {
				case <-ctx.Done():
					stabTimer.Stop()
					return

				case <-resetCh:
					// Restart entire cycle.
					stabTimer.Stop()
					goto restart

				case <-stabTimer.C:
					// Stability achieved.
				}
			}

			// Emit event unless shutting down.
			select {
			case <-ctx.Done():
				return
			case events <- struct{}{}:
			}

			// Reset state.
			timer = nil
			timerCh = nil
		}
	}
}
