// Package retention promotes snapshots into rule folders and prunes old ones.
package retention

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/robfig/cron/v3"
)

// Engine manages promotion and cleanup rules.
type Engine struct {
	mu  sync.RWMutex
	cfg Config
	log logging.Logger
}

type Rule struct {
	Name  string `yaml:"name"`
	Cron  string `yaml:"cron"`
	Count int    `yaml:"count"`
}

// New creates a retention engine from cfg.
func New(log logging.Logger) *Engine {
	return &Engine{
		cfg: Config{},
		log: log,
	}
}

// UpdateConfig hot‑reloads retention rules.
func (e *Engine) UpdateConfig(config Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg = config
}

// Apply promotes the new snapshotwatcher and prunes old ones.
func (e *Engine) Apply(ctx context.Context, archiveRoot, newSnapshotDir string) error {
	e.mu.RLock()
	rules := append([]Rule(nil), e.cfg.Rules...)
	removeUnknownFolders := e.cfg.RemoveUnknownFolders
	e.mu.RUnlock()

	ts, err := parseTimestamp(filepath.Base(newSnapshotDir))
	if err != nil {
		return fmt.Errorf("invalid snapshotwatcher timestamp: %w", err)
	}

	for _, rule := range rules {
		ruleDir := filepath.Join(archiveRoot, rule.Name)

		if strings.TrimSpace(rule.Cron) != "" {
			if err := e.promote(rule, ruleDir, newSnapshotDir, ts); err != nil {
				e.log.Error("promote failed", "goPackage", "retention", "ruleName", rule.Name, "error", err)
			}
		}

		if err := e.cleanup(rule, ruleDir); err != nil {
			e.log.Error("retention - cleanup %s failed", "goPackage", "retention", "ruleName", rule.Name, "error", err)
		}
	}

	if removeUnknownFolders {
		if err := e.removeUnknownFolders(rules, archiveRoot); err != nil {
			e.log.Error("retention - remove unknown folders failed", "goPackage", "retention", "error", err)
		}
	}

	return nil
}

// promote copies the snapshotwatcher if none exists after the cron boundary.
func (e *Engine) promote(rule Rule, ruleDir, snapDir string, snapTS time.Time) error {
	sched, err := cron.ParseStandard(rule.Cron)
	if err != nil {
		return fmt.Errorf("invalid cron %q: %w", rule.Cron, err)
	}

	// Compute last cron boundary.
	prev := prevCron(sched, snapTS)
	next := sched.Next(prev)

	// Ensure folder exists.
	if err := os.MkdirAll(ruleDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Check if a snapshotwatcher already exists after boundary.
	existing, err := listSnapshotDirs(ruleDir)
	if err != nil {
		return err
	}

	// If a snapshot already exists in this cron window, skip promotion.
	for _, ts := range existing {
		if !ts.Before(prev) && ts.Before(next) {
			return nil
		}
	}

	// Promote by copying the directory.
	dst := filepath.Join(ruleDir, filepath.Base(snapDir))
	return copyDir(snapDir, dst)
}

// cleanup keeps only the newest N snapshotwatcher directories.
func (e *Engine) cleanup(rule Rule, ruleDir string) error {
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

// removeUnknownFolders removes folders that are not defined in the retention rules.
func (e *Engine) removeUnknownFolders(rules []Rule, ruleDir string) error {
	known := make(map[string]struct{})
	for _, r := range rules {
		known[r.Name] = struct{}{}
	}

	entries, err := os.ReadDir(ruleDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		if _, ok := known[name]; !ok {
			full := filepath.Join(ruleDir, name)
			e.log.Warn("Removing unknown folder", "path", full)
			if err := os.RemoveAll(full); err != nil {
				return fmt.Errorf("removing dir %s: %w", full, err)
			}
		}
	}

	return nil
}

// listSnapshotDirs returns timestamps of snapshotwatcher directories.
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

// parseTimestamp parses snapshotwatcher directory names.
func parseTimestamp(name string) (time.Time, error) {
	return time.Parse("2006-01-02T15-04-05", name)
}

// copyDir copies a snapshotwatcher directory recursively.
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
	// Start far enough in the past to guarantee we cross the boundary
	cur := t.Add(-48 * time.Hour)

	// Move forward until the next boundary is >= t
	for {
		next := s.Next(cur)
		if !next.Before(t) {
			return cur
		}
		cur = next
	}
}

func sameWindow(ts, boundary time.Time, sched cron.Schedule) bool {
	// next boundary after "boundary"
	next := sched.Next(boundary)

	// ts must be >= boundary AND < next boundary
	return !ts.Before(boundary) && ts.Before(next)
}
