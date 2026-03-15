package snapshotwatcher

import (
	"sort"
	"strings"
)

// UpdateConfig hot‑reloads source directory and file names.
func (sw *Watcher) UpdateConfig(cfg Config) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	if sw.isSameConfig(cfg, sw.cfg) {
		sw.logg.Debug("same config, returning", "function", "UpdateConfig")
		return
	}
	sw.cfg = cfg
	sw.logg.Info("config updated")
}

// NeedsRestart reports whether watcher must be restarted for config change.
func (sw *Watcher) NeedsRestart(newCfg Config) bool {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	oldCfg := sw.cfg
	sw.logg.Debug("checking if config needs restart", "function", "NeedsRestart")
	return !sw.isSameConfig(oldCfg, newCfg)
}

func (sw *Watcher) isSameConfig(oldCfg, newCfg Config) bool {
	// Compare simple fields
	if oldCfg.WatchMode != newCfg.WatchMode ||
		oldCfg.Path != newCfg.Path ||
		oldCfg.PrimaryName != newCfg.PrimaryName {
		sw.logg.Debug("different configs - fields", "function", "isSameConfig")
		return false
	}

	// Compare AuxNames (order-insensitive)
	if len(oldCfg.AuxNames) != len(newCfg.AuxNames) {
		sw.logg.Debug("different configs - aux names count", "function", "isSameConfig")
		return false
	}

	// Sort copies of AuxNames
	oldAux := append([]string(nil), oldCfg.AuxNames...)
	newAux := append([]string(nil), newCfg.AuxNames...)
	sort.Strings(oldAux)
	sort.Strings(newAux)

	// Join with | and compare
	oldJoined := strings.Join(oldAux, "|")
	newJoined := strings.Join(newAux, "|")

	if oldJoined != newJoined {
		sw.logg.Debug("different configs - aux names", "function", "isSameConfig")
		return false
	}

	return true
}
