// Package worker processes snapshotwatcher jobs and writes atomic snapshotwatcher directories.
package worker

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/raoulx24/rdb-archiver/internal/fs"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/mailbox"
	"github.com/raoulx24/rdb-archiver/internal/retention"
	"github.com/raoulx24/rdb-archiver/internal/snapshot"
)

// Worker writes snapshots into destination folders and applies retention.
type Worker struct {
	mu        sync.RWMutex
	cfg       Config
	fs        fs.FS
	logg      logging.Logger
	retention *retention.Retention
	mb        *mailbox.Mailbox[snapshot.Job]
}

// New creates a worker using destination config and mailbox.
func New(cfg Config, log logging.Logger, r *retention.Retention, mb *mailbox.Mailbox[snapshot.Job], filesystem fs.FS) *Worker {
	logg := log.With("pkg", "worker")
	logg.Debug("creating worker")
	return &Worker{
		cfg:       cfg,
		fs:        filesystem,
		logg:      logg,
		retention: r,
		mb:        mb,
	}
}

// UpdateConfig hot‑reloads destination settings.

// Start runs the worker loop using mailbox semantics.
func (w *Worker) Start(ctx context.Context) {
	w.logg.Info("starting worker")
	w.updateRetentionRules()
	for {
		job, ok := w.mb.Take(ctx)
		if !ok {
			w.logg.Info("worker stopped")
			return
		}
		if err := w.Handle(ctx, job.Snap); err != nil {
			w.logg.Error("snapshot handle failed", "error", err)
		}
	}
}

// Handle writes a snapshotwatcher directory and applies retention.
func (w *Worker) Handle(ctx context.Context, snap snapshot.Snapshot) error {
	w.logg.Debug("worker starting snapshot handling")
	finalDir, err := w.writeSnapshot(ctx, snap)
	if err != nil {
		return err
	}

	w.mu.RLock()
	dest := w.cfg
	w.mu.RUnlock()

	root := filepath.Join(dest.Root, dest.SubDir)
	w.logg.Debug("destination root resolved", "root", root)

	if err := w.retention.Apply(ctx, w.fs, root, finalDir); err != nil {
		w.logg.Error("worker: retention failed", "error", err)
	}

	return nil
}

func (w *Worker) UpdateConfig(cfg Config) {
	w.logg.Debug("uppdating config")
	w.mu.Lock()
	w.cfg = cfg
	w.mu.Unlock()

	w.updateRetentionRules()
}

// writeSnapshot copies all snapshotwatcher files into an atomic directory.
func (w *Worker) writeSnapshot(ctx context.Context, snap snapshot.Snapshot) (string, error) {
	w.mu.RLock()
	dest := w.cfg
	w.mu.RUnlock()

	root := filepath.Join(dest.Root, dest.SubDir)
	lastDir := filepath.Join(root, dest.SnapshotSubdir)

	//ts := time.Now().UTC().Format("2006-01-02T15-04-05")
	ts := snap.Primary.ModTime.UTC().Format("2006-01-02T15-04-05")
	tmpDir := filepath.Join(lastDir, ".tmp-"+ts)
	finalDir := filepath.Join(lastDir, ts)
	w.logg.Debug("new destinations", "tmpDir", tmpDir, "finalDir", finalDir)

	if err := w.fs.MkdirAll(tmpDir); err != nil {
		return "", fmt.Errorf("creating tmp dir: %w", err)
	}

	// Copy primary
	if err := w.copyArtifact(ctx, snap.Primary, snap.Dir, tmpDir); err != nil {
		_ = w.fs.RemoveAll(tmpDir)
		return "", err
	}

	// Copy aux
	for _, a := range snap.Aux {
		if err := w.copyArtifact(ctx, a, snap.Dir, tmpDir); err != nil {
			_ = w.fs.RemoveAll(tmpDir)
			return "", err
		}
	}

	// Finalize atomically
	if _, err := w.fs.Stat(finalDir); err == nil {
		if err := w.fs.RemoveAll(finalDir); err != nil {
			return "", fmt.Errorf("failed to remove existing finalDir: %w", err)
		}
	}
	if err := w.fs.Rename(ctx, tmpDir, finalDir); err != nil {
		_ = w.fs.RemoveAll(tmpDir)
		return "", fmt.Errorf("finalizing snapshotwatcher: %w", err)
	}

	return finalDir, nil
}

// copyArtifact copies one file into the snapshotwatcher directory.
func (w *Worker) copyArtifact(ctx context.Context, a snapshot.Artifact, srcDir string, dstDir string) error {
	src := filepath.Join(srcDir, a.Name)
	dst := filepath.Join(dstDir, a.Name)
	if err := w.fs.CopyFile(ctx, src, dst); err != nil {
		return fmt.Errorf("copying %s: %w", a.Name, err)
	}
	w.logg.Info("worker copied artifact", "artifact", a.Name, "src", src, "dst", dst)
	return nil
}

// updateRetentionRules adds to the retention rules the snapshotwatcher one
func (w *Worker) updateRetentionRules() {
	w.logg.Debug("entering Worker.updateRetentionRules")
	w.mu.RLock()
	mainRule := retention.Rule{
		Name:  w.cfg.SnapshotSubdir,
		Cron:  "",
		Count: w.cfg.Retention.LastCount,
	}
	updated := append([]retention.Rule{mainRule}, w.cfg.Retention.Rules...)
	removeUnknownFolders := w.cfg.Retention.RemoveUnknownFolders
	w.mu.RUnlock()
	w.retention.UpdateConfig(retention.Config{RemoveUnknownFolders: removeUnknownFolders, Rules: updated})
}
