// Package watchfs provides file‑change detection via fsnotify or polling.
package watchfs

import (
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/logging"
)

// FileWatcher emits change events for a file using fsnotify or polling
type FileWatcher struct {
	mu              sync.RWMutex
	cfg             Config
	logg            logging.Logger
	debounceWindow  time.Duration
	stabilityWindow time.Duration
	pollInterval    time.Duration
	lastModTime     map[string]time.Time
}

// New creates a FileWatcher from config values.
func New(cfg Config, log logging.Logger) (*FileWatcher, error) {
	logg := log.With("pkg", "watchfs")
	logg.Debug("creating watch fs")

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
		logg:            logg,
		debounceWindow:  debounce,
		pollInterval:    interval,
		stabilityWindow: stability,
		lastModTime:     make(map[string]time.Time),
	}, nil
}

// UpdateConfig hot‑reloads timing parameters safely.
func (wfs *FileWatcher) UpdateConfig(cfg Config) error {
	wfs.logg.Debug("updating config")
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

	wfs.mu.Lock()
	wfs.cfg = cfg
	wfs.debounceWindow = debounce
	wfs.pollInterval = interval
	wfs.stabilityWindow = stability
	wfs.mu.Unlock()

	return nil
}
