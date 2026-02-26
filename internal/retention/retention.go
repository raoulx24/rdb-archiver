// Package retention promotes snapshots into rule folders and prunes old ones.
package retention

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/robfig/cron/v3"
)

// Engine manages promotion and cleanup rules.
type Engine struct {
	mu    sync.RWMutex
	rules []config.RetentionRule
	log   logging.Logger
}

// New creates a retention engine from config.
func New(cfg *config.Config, log logging.Logger) *Engine {
	return &Engine{
		rules: cfg.Destination.Retention.Rules,
		log:   log,
	}
}

// UpdateConfig hot‑reloads retention rules.
func (e *Engine) UpdateConfig(cfg *config.Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = cfg.Destination.Retention.Rules
}

// Apply promotes the new snapshot and prunes old ones.
func (e *Engine) Apply(ctx context.Context, archiveRoot, newSnapshotDir string) error {
	e.mu.RLock()
	rules := append([]config.RetentionRule(nil), e.rules...)
	e.mu.RUnlock()

	ts, err := parseTimestamp(filepath.Base(newSnapshotDir))
	if err != nil {
		return fmt.Errorf("invalid snapshot timestamp: %w", err)
	}

	for _, rule := range rules {
		ruleDir := filepath.Join(archiveRoot, rule.Name)

		if err := e.promote(rule, ruleDir, newSnapshotDir, ts); err != nil {
			e.log.Error("retention: promote %s failed: %v", rule.Name, err)
		}

		if err := e.cleanup(rule, ruleDir); err != nil {
			e.log.Error("retention: cleanup %s failed: %v", rule.Name, err)
		}
	}

	return nil
}

// promote copies the snapshot if none exists after the cron boundary.
func (e *Engine) promote(rule config.RetentionRule, ruleDir, snapDir string, snapTS time.Time) error {
	sched, err := cron.ParseStandard(rule.Cron)
	if err != nil {
		return fmt.Errorf("invalid cron %q: %w", rule.Cron, err)
	}

	// Compute last cron boundary.
	prev := prevCron(sched, snapTS)

	// Ensure folder exists.
	if err := os.MkdirAll(ruleDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Check if a snapshot already exists after boundary.
	existing, err := listSnapshotDirs(ruleDir)
	if err != nil {
		return err
	}
	for _, ts := range existing {
		if ts.After(prev) {
			return nil // already promoted
		}
	}

	// Promote by copying the directory.
	dst := filepath.Join(ruleDir, filepath.Base(snapDir))
	return copyDir(snapDir, dst)
}

// cleanup keeps only the newest N snapshot directories.
func (e *Engine) cleanup(rule config.RetentionRule, ruleDir string) error {
	entries, err := os.ReadDir(ruleDir)
	if err != nil {
		return fmt.Errorf("reading folder: %w", err)
	}

	var dirs []string
	for _, ent := range entries {
		if ent.IsDir() {
			dirs = append(dirs, ent.Name())
		}
	}

	if len(dirs) <= rule.Count {
		return nil
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i] > dirs[j] // lexicographic works for your timestamp format
	})

	for _, name := range dirs[rule.Count:] {
		_ = os.RemoveAll(filepath.Join(ruleDir, name))
	}

	return nil
}

// listSnapshotDirs returns timestamps of snapshot directories.
func listSnapshotDirs(dir string) ([]time.Time, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var out []time.Time
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		ts, err := parseTimestamp(ent.Name())
		if err == nil {
			out = append(out, ts)
		}
	}
	return out, nil
}

// parseTimestamp parses snapshot directory names.
func parseTimestamp(name string) (time.Time, error) {
	return time.Parse("2006-01-02T15-04-05", name)
}

// copyDir copies a snapshot directory recursively.
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, ent := range entries {
		s := filepath.Join(src, ent.Name())
		d := filepath.Join(dst, ent.Name())

		if ent.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
			continue
		}

		if err := copyFile(s, d); err != nil {
			return err
		}
	}

	return nil
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// prevCron returns the most recent cron boundary before t.
func prevCron(s cron.Schedule, t time.Time) time.Time {
	// Start slightly before t to avoid returning t itself.
	cur := t.Add(-time.Second)

	// Move backwards until s.Next(prev) >= t.
	for {
		next := s.Next(cur)
		if !next.Before(t) {
			return cur
		}
		cur = cur.Add(-time.Minute)
	}
}
