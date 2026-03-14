package snapshotwatcher

import (
	"sort"
	"strings"
)

// UpdateConfig hot‑reloads source directory and file names.
func (sw *Watcher) UpdateConfig(cfg Config) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	if isSameConfig(cfg, sw.cfg) {
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
	return isSameConfig(oldCfg, newCfg)
}

func isSameConfig(oldCfg, newCfg Config) bool {
	// Compare simple fields
	if oldCfg.WatchMode != newCfg.WatchMode ||
		oldCfg.Path != newCfg.Path ||
		oldCfg.PrimaryName != newCfg.PrimaryName {
		return false
	}

	// Compare AuxNames (order-insensitive)
	if len(oldCfg.AuxNames) != len(newCfg.AuxNames) {
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

	return oldJoined == newJoined
}
