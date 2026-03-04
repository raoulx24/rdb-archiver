package snapshotwatcher

// UpdateConfig hot‑reloads source directory and file names.
func (sw *Watcher) UpdateConfig(cfg Config) {
	sw.mu.Lock()
	sw.cfg = cfg
	sw.mu.Unlock()
}

// NeedsRestart reports whether watcher must be restarted for config change.
func (sw *Watcher) NeedsRestart(oldCfg, newCfg Config) bool {
	return oldCfg.WatchMode != newCfg.WatchMode ||
		oldCfg.Path != newCfg.Path ||
		oldCfg.PrimaryName != newCfg.PrimaryName
}
