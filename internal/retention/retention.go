// Package retention promotes snapshots into rule folders and prunes old ones.
package retention

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/fs"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/robfig/cron/v3"
)

// Retention manages promotion and cleanup rules.
type Retention struct {
	mu   sync.RWMutex
	fs   fs.FS
	cfg  Config
	logg logging.Logger
}

type Rule struct {
	Name  string `yaml:"name"`
	Cron  string `yaml:"cron"`
	Count int    `yaml:"count"`
}

// New creates a retention engine from cfg.
func New(logg logging.Logger) *Retention {
	return &Retention{
		cfg:  Config{},
		logg: logg,
	}
}

// UpdateConfig hot‑reloads retention rules.
func (r *Retention) UpdateConfig(config Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cfg = config
}

// Apply promotes the new snapshotwatcher and prunes old ones.
func (r *Retention) Apply(ctx context.Context, filesystem fs.FS, archiveRoot, newSnapshotDir string) error {
	r.logg.Info("retention engine is starting to apply rules")
	r.fs = filesystem
	r.mu.RLock()
	rules := append([]Rule(nil), r.cfg.Rules...)
	removeUnknownFolders := r.cfg.RemoveUnknownFolders
	r.mu.RUnlock()

	ts, err := parseTimestamp(filepath.Base(newSnapshotDir))
	if err != nil {
		return fmt.Errorf("invalid snapshotwatcher timestamp: %w", err)
	}

	for _, rule := range rules {
		ruleDir := filepath.Join(archiveRoot, rule.Name)

		if strings.TrimSpace(rule.Cron) != "" {
			if err := r.promote(ctx, rule, ruleDir, newSnapshotDir, ts); err != nil {
				r.logg.Error("promote failed", "goPackage", "retention", "ruleName", rule.Name, "error", err)
			}
		}

		if err := r.cleanup(rule, ruleDir); err != nil {
			r.logg.Error("retention - cleanup %s failed", "goPackage", "retention", "ruleName", rule.Name, "error", err)
		}
	}

	if removeUnknownFolders {
		if err := r.removeUnknownFolders(rules, archiveRoot); err != nil {
			r.logg.Error("retention - remove unknown folders failed", "goPackage", "retention", "error", err)
		}
	}

	return nil
}

// promote copies the snapshotwatcher if none exists after the cron boundary.
func (r *Retention) promote(ctx context.Context, rule Rule, ruleDir, snapDir string, snapTS time.Time) error {
	sched, err := cron.ParseStandard(rule.Cron)
	if err != nil {
		return fmt.Errorf("invalid cron %q: %w", rule.Cron, err)
	}

	// Compute last cron boundary.
	prev := prevCron(sched, snapTS)
	next := sched.Next(prev)

	// Ensure folder exists.
	if err := r.fs.MkdirAll(ruleDir); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Check if a snapshotwatcher already exists after boundary.
	existing, err := r.listSnapshotDirs(ruleDir)
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
	r.logg.Info("creating snapshot in cron folder", "rule", rule.Name, "cron", rule.Cron, "snapshot", filepath.Base(snapDir))
	return r.fs.CopyDir(ctx, snapDir, dst)
}

// cleanup keeps only the newest N snapshotwatcher directories.
func (r *Retention) cleanup(rule Rule, ruleDir string) error {
	entries, err := r.fs.ReadDir(ruleDir)
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
		r.logg.Info("removing old snapshot in cron folder", "rule", rule.Name, "cron", rule.Cron, "snapshot", name)
		_ = r.fs.RemoveAll(filepath.Join(ruleDir, name))
	}

	return nil
}

// removeUnknownFolders removes folders that are not defined in the retention rules.
func (r *Retention) removeUnknownFolders(rules []Rule, ruleDir string) error {
	known := make(map[string]struct{})
	for _, r := range rules {
		known[r.Name] = struct{}{}
	}

	entries, err := r.fs.ReadDir(ruleDir)
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
			r.logg.Warn("Removing unknown cron folder", "path", full)
			if err := r.fs.RemoveAll(full); err != nil {
				return fmt.Errorf("removing dir %s: %w", full, err)
			}
		}
	}

	return nil
}

// listSnapshotDirs returns timestamps of snapshotwatcher directories.
func (r *Retention) listSnapshotDirs(dir string) ([]time.Time, error) {
	entries, err := r.fs.ReadDir(dir)
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
