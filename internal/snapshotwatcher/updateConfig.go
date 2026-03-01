package snapshotwatcher

// UpdateConfig hot‑reloads source directory and file names.
func (w *SnapshotWatcher) UpdateConfig(cfg Config) {
	w.mu.Lock()
	w.cfg = cfg
	w.mu.Unlock()
}
