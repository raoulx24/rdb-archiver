package retention

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

type Config struct {
	RemoveUnknownFolders bool
	Rules                []Rule
	SnapshotExtension    string
}

// isSameConfig compares the configs
func (r *Retention) isSameConfig(oldConfig Config, newConfig Config) bool {
	if oldConfig.RemoveUnknownFolders != newConfig.RemoveUnknownFolders ||
		oldConfig.SnapshotExtension != newConfig.SnapshotExtension {
		r.logg.Debug("different configs - fields", "function", "isSameConfig")
		return false
	}

	if len(oldConfig.Rules) != len(newConfig.Rules) {
		r.logg.Debug("different configs - rules length", "function", "isSameConfig")
		return false
	}

	hashRules := func(rules []Rule) map[string]struct{} {
		m := make(map[string]struct{}, len(rules))
		for _, r := range rules {
			// Trim strings
			name := strings.TrimSpace(r.Name)
			cron := strings.TrimSpace(r.Cron)
			count := strconv.Itoa(r.Count)

			// Concatenate fields for hashing
			data := name + "|" + cron + "|" + count
			sum := sha256.Sum256([]byte(data))
			m[hex.EncodeToString(sum[:])] = struct{}{}
		}
		return m
	}

	oldMap := hashRules(oldConfig.Rules)
	newMap := hashRules(newConfig.Rules)

	// maybe there were duplicates before
	if len(oldMap) != len(newMap) {
		r.logg.Debug("different configs - rules hash length", "function", "isSameConfig")
		return false
	}

	for h := range oldMap {
		if _, ok := newMap[h]; !ok {
			r.logg.Debug("different configs - rules hash mismatch", "function", "isSameConfig")
			return false
		}
	}

	r.logg.Debug("same config", "function", "isSameConfig")
	return true
}
