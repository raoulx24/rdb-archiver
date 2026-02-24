// Package retention implements event-driven, cron-anchored snapshot retention.
package retention

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/snapshot"
	"github.com/robfig/cron/v3"
)

// Engine applies retention rules to snapshots in the archive directory.
type Engine struct {
	cfg config.Config
	log logging.Logger
}

func New(cfg config.Config, log logging.Logger) *Engine {
	return &Engine{cfg: cfg, log: log}
}

// Apply runs retention after a new snapshot is created.
// newSnapPath is the full path of the newly archived snapshot.
func (e *Engine) Apply(ctx context.Context, archiveDir, newSnapPath string) error {
	snaps, err := loadSnapshots(archiveDir)
	if err != nil {
		return err
	}

	// Sort newest → oldest
	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].Timestamp.After(snaps[j].Timestamp)
	})

	newSnap := findSnapshot(snaps, newSnapPath)
	if newSnap == nil {
		return nil
	}

	// Apply cron-based rules (promotion logic only for now)
	for _, rule := range e.cfg.Retention.Rules {
		if err := e.applyRule(ctx, snaps, newSnap, rule); err != nil {
			e.log.Error("retention rule %s failed: %v", rule.Name, err)
		}
	}

	// Apply global last_count
	if err := e.applyLastCount(ctx, snaps); err != nil {
		e.log.Error("retention last_count failed: %v", err)
	}

	return nil
}

func (e *Engine) applyRule(
	ctx context.Context,
	snaps []snapshot.Snapshot,
	newSnap *snapshot.Snapshot,
	rule config.RetentionRule,
) error {
	// Parse cron expression
	schedule, err := cron.ParseStandard(rule.Cron)
	if err != nil {
		return err
	}

	now := time.Now()

	// Find most recent anchor BEFORE now
	// We walk forward from a window in the past to now.
	start := now.Add(-30 * 24 * time.Hour)
	t := schedule.Next(start)
	prev := t
	for t.Before(now) || t.Equal(now) {
		prev = t
		t = schedule.Next(t)
	}
	anchor := prev

	// Find eligible snapshots: AFTER anchor
	var eligible []snapshot.Snapshot
	for _, s := range snaps {
		if s.Timestamp.After(anchor) {
			eligible = append(eligible, s)
		}
	}
	if len(eligible) == 0 {
		return nil
	}

	// Find snapshot closest to anchor (but after it)
	winner := &eligible[0]
	for i := 1; i < len(eligible); i++ {
		dCur := eligible[i].Timestamp.Sub(anchor)
		dWin := winner.Timestamp.Sub(anchor)
		if dCur < dWin {
			winner = &eligible[i]
		}
	}

	// Promote only if new snapshot is the winner
	if winner.Path != newSnap.Path {
		return nil
	}

	e.log.Info("retention: rule %s matched snapshot %s", rule.Name, newSnap.Path)
	// Tier-specific pruning can be added later once naming/tiering is defined.
	return nil
}

func (e *Engine) applyLastCount(ctx context.Context, snaps []snapshot.Snapshot) error {
	keep := e.cfg.Retention.LastCount
	if keep <= 0 || len(snaps) <= keep {
		return nil
	}

	toDelete := snaps[keep:]
	for _, s := range toDelete {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := os.Remove(s.Path); err != nil {
			e.log.Error("retention: failed to remove %s: %v", s.Path, err)
		} else {
			e.log.Info("retention: removed %s", s.Path)
		}
	}

	return nil
}

func loadSnapshots(dir string) ([]snapshot.Snapshot, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var snaps []snapshot.Snapshot
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		full := filepath.Join(dir, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}

		snaps = append(snaps, snapshot.FromFileInfo(full, info))
	}

	return snaps, nil
}

func findSnapshot(snaps []snapshot.Snapshot, path string) *snapshot.Snapshot {
	for i := range snaps {
		if snaps[i].Path == path {
			return &snaps[i]
		}
	}
	return nil
}
