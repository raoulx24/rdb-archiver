package snapshotwatcher

// UpdateConfig hot‑reloads source directory and file names.
func (w *SnapshotWatcher) UpdateConfig(cfg Config) {
	w.mu.Lock()
	w.cfg = cfg
	w.mu.Unlock()
}

// NeedsRestart reports whether watcher must be restarted for config change.
func (w *SnapshotWatcher) NeedsRestart(oldCfg, newCfg Config) bool {
	return oldCfg.WatchMode != newCfg.WatchMode ||
		oldCfg.Path != newCfg.Path ||
		oldCfg.PrimaryName != newCfg.PrimaryName
}
