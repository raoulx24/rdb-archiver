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
}

// isSameConfig compares the configs
func isSameConfig(oldConfig Config, newConfig Config) bool {
	if oldConfig.RemoveUnknownFolders != newConfig.RemoveUnknownFolders {
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

	if len(oldMap) != len(newMap) {
		return false
	}

	for h := range oldMap {
		if _, ok := newMap[h]; !ok {
			return false
		}
	}

	return true
}
