package retention

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/logging"
)

type Engine struct {
	rules []config.RetentionRule
	log   logging.Logger
}

func New(cfg *config.Config, log logging.Logger) *Engine {
	return &Engine{
		rules: cfg.Retention.Rules, // ✔ correct field
		log:   log,
	}
}

// SnapshotEvent represents a paired dump+nodes snapshot.
type SnapshotEvent struct {
	Timestamp time.Time
	DumpPath  string
	NodesPath string
}

// Apply runs retention for all rules.
func (e *Engine) Apply(ctx context.Context, archiveRoot string, newSnapshotPath string) error {
	for _, rule := range e.rules {
		folder := filepath.Join(archiveRoot, rule.Name)
		if err := e.applyRule(ctx, folder, rule.Count); err != nil {
			e.log.Error("retention: rule %s failed: %v", rule.Name, err)
		}
	}
	return nil
}

// applyRule keeps only the last N snapshot events in a folder.
func (e *Engine) applyRule(ctx context.Context, folder string, keep int) error {
	events, err := scanSnapshotEvents(folder)
	if err != nil {
		return err
	}

	if len(events) <= keep {
		return nil
	}

	// Sort newest → oldest
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})

	toDelete := events[keep:]

	for _, ev := range toDelete {
		if ev.DumpPath != "" {
			_ = os.Remove(ev.DumpPath)
		}
		if ev.NodesPath != "" {
			_ = os.Remove(ev.NodesPath)
		}
	}

	return nil
}

// scanSnapshotEvents finds dump+nodes pairs in a folder.
func scanSnapshotEvents(folder string) ([]SnapshotEvent, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, fmt.Errorf("reading folder: %w", err)
	}

	dumpFiles := map[string]string{}  // ts → dump path
	nodesFiles := map[string]string{} // ts → nodes path

	for _, ent := range entries {
		name := ent.Name()
		full := filepath.Join(folder, name)

		if strings.HasPrefix(name, "dump-") && strings.HasSuffix(name, ".rdb") {
			ts := extractTimestamp(name, "dump-", ".rdb")
			if ts != "" {
				dumpFiles[ts] = full
			}
		}

		if strings.HasPrefix(name, "nodes-") && strings.HasSuffix(name, ".config") {
			ts := extractTimestamp(name, "nodes-", ".config")
			if ts != "" {
				nodesFiles[ts] = full
			}
		}
	}

	// Build events
	var events []SnapshotEvent
	for ts, dumpPath := range dumpFiles {
		t, err := time.Parse("2006-01-02T15-04-05", ts)
		if err != nil {
			continue
		}

		ev := SnapshotEvent{
			Timestamp: t,
			DumpPath:  dumpPath,
			NodesPath: nodesFiles[ts], // may be empty if missing
		}

		events = append(events, ev)
	}

	return events, nil
}

// extractTimestamp removes prefix/suffix and returns the timestamp string.
func extractTimestamp(name, prefix, suffix string) string {
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
		return ""
	}
	core := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
	return core
}
