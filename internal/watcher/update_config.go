package watcher

import (
	"time"

	"github.com/raoulx24/rdb-archiver/internal/config"
)

// UpdateConfig updates watcher fields atomically for hot‑reload.
func (w *Watcher) UpdateConfig(cfg config.SourceConfig) {
	w.mu.Lock()
	defer w.mu.Unlock()

	primaryChanged := cfg.Path != w.dir || cfg.PrimaryName != w.primaryName

	w.dir = cfg.Path
	w.primaryName = cfg.PrimaryName
	w.auxNames = cfg.AuxNames
	w.interval = cfg.Watch.PollInterval
	w.mode = cfg.Watch.Mode
	w.debounce = cfg.Watch.DebounceWindow

	if primaryChanged {
		w.lastModTime = time.Time{}
	}
}
