// Package retention implements snapshot retention policies for archived files.
package retention

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/snapshot"
)

// Engine defines the interface for applying retention policies to an archive directory.
type Engine interface {
	Apply(ctx context.Context, archiveDir string) error
}

// KeepLastN is a simple retention engine that keeps the N most recent snapshots
// and deletes the rest.
type KeepLastN struct {
	N      int
	Logger logging.Logger
}

// NewKeepLastN creates a new KeepLastN retention engine.
func NewKeepLastN(n int, logger logging.Logger) *KeepLastN {
	return &KeepLastN{
		N:      n,
		Logger: logger,
	}
}

// Apply scans the archive directory, sorts snapshots by timestamp, and deletes
// all but the N most recent ones.
func (k *KeepLastN) Apply(ctx context.Context, archiveDir string) error {
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		return err
	}

	var snaps []snapshot.Snapshot
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		full := filepath.Join(archiveDir, e.Name())
		info, err := e.Info()
		if err != nil {
			k.Logger.Error("retention: stat failed for %s: %v", full, err)
			continue
		}

		snaps = append(snaps, snapshot.FromFileInfo(full, info))
	}

	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].Timestamp.After(snaps[j].Timestamp)
	})

	if len(snaps) <= k.N {
		return nil
	}

	toDelete := snaps[k.N:]
	for _, s := range toDelete {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := os.Remove(s.Path); err != nil {
			k.Logger.Error("retention: failed to remove %s: %v", s.Path, err)
			continue
		}
		k.Logger.Info("retention: removed %s", s.Path)
	}

	return nil
}
