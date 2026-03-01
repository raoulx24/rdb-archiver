// Package watchfs provides file‑change detection via fsnotify or polling.
package watchfs

import (
	"sync"
	"time"
)

// FileWatcher emits change events for a file using fsnotify or polling.
type FileWatcher struct {
	mu              sync.RWMutex
	cfg             Config
	debounceWindow  time.Duration
	stabilityWindow time.Duration
	pollInterval    time.Duration
}

// New creates a FileWatcher from config values.
func New(cfg Config) (*FileWatcher, error) {
	debounce, err := time.ParseDuration(cfg.FSNotify.DebounceWindow)
	if err != nil {
		return nil, err
	}

	interval, err := time.ParseDuration(cfg.Pool.Interval)
	if err != nil {
		return nil, err
	}

	stability, err := time.ParseDuration(cfg.StabilityWindow)
	if err != nil {
		return nil, err
	}

	return &FileWatcher{
		cfg:             cfg,
		debounceWindow:  debounce,
		pollInterval:    interval,
		stabilityWindow: stability,
	}, nil
}

// UpdateConfig hot‑reloads timing parameters safely.
func (w *FileWatcher) UpdateConfig(cfg Config) error {
	debounce, err := time.ParseDuration(cfg.FSNotify.DebounceWindow)
	if err != nil {
		return err
	}

	interval, err := time.ParseDuration(cfg.Pool.Interval)
	if err != nil {
		return err
	}

	stability, err := time.ParseDuration(cfg.StabilityWindow)
	if err != nil {
		return err
	}

	w.mu.Lock()
	w.cfg = cfg
	w.debounceWindow = debounce
	w.pollInterval = interval
	w.stabilityWindow = stability
	w.mu.Unlock()

	return nil
}
