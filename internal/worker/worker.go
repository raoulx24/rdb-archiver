// Package worker processes snapshot jobs and writes atomic snapshot directories.
package worker

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/fs"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/mailbox"
	"github.com/raoulx24/rdb-archiver/internal/retention"
	"github.com/raoulx24/rdb-archiver/internal/snapshot"
)

// Worker writes snapshots into destination folders and applies retention.
type Worker struct {
	mu        sync.RWMutex
	dest      config.DestinationConfig
	fs        fs.FS
	log       logging.Logger
	retention *retention.Engine
	mb        *mailbox.Mailbox[Job]
}

// New creates a worker using destination config and mailbox.
func New(dest config.DestinationConfig, log logging.Logger, r *retention.Engine, mb *mailbox.Mailbox[Job], filesystem fs.FS) *Worker {
	log.Debug("creating worker")
	if filesystem == nil {
		filesystem = fs.New()
	}
	return &Worker{
		dest:      dest,
		fs:        filesystem,
		log:       log,
		retention: r,
		mb:        mb,
	}
}

// UpdateConfig hot‑reloads destination settings.

// Start runs the worker loop using mailbox semantics.
func (w *Worker) Start(ctx context.Context) {
	w.log.Info("starting worker")
	w.updateRetentionRules()
	for {
		job := w.mb.Take()
		if err := w.Handle(ctx, job.Snap); err != nil {
			w.log.Error("worker: snapshot failed", "error", err)
		}
	}
}

// Handle writes a snapshot directory and applies retention.
func (w *Worker) Handle(ctx context.Context, snap snapshot.Snapshot) error {
	w.log.Debug("entering Worker.Handle()")
	finalDir, err := w.writeSnapshot(ctx, snap)
	if err != nil {
		return err
	}

	w.mu.RLock()
	dest := w.dest
	w.mu.RUnlock()

	root := filepath.Join(dest.Root, dest.SubDir)
	w.log.Debug("destination root resolved", "root", root)

	if err := w.retention.Apply(ctx, root, finalDir); err != nil {
		w.log.Error("worker: retention failed", "error", err)
	}

	return nil
}

func (w *Worker) UpdateConfig(dest config.DestinationConfig) {
	w.log.Debug("entering Worker.UpdateConfig()")
	w.mu.Lock()
	w.dest = dest
	w.mu.Unlock()

	w.updateRetentionRules()
}

// writeSnapshot copies all snapshot files into an atomic directory.
func (w *Worker) writeSnapshot(ctx context.Context, snap snapshot.Snapshot) (string, error) {
	w.log.Debug("entering Worker.writeSnapshot()")
	w.mu.RLock()
	dest := w.dest
	w.mu.RUnlock()

	root := filepath.Join(dest.Root, dest.SubDir)
	lastDir := filepath.Join(root, dest.SnapshotSubdir)

	//ts := time.Now().UTC().Format("2006-01-02T15-04-05")
	ts := snap.Primary.ModTime.UTC().Format("2006-01-02T15-04-05")
	tmpDir := filepath.Join(lastDir, ".tmp-"+ts)
	finalDir := filepath.Join(lastDir, ts)
	w.log.Debug("new destinations", "tmpDir", tmpDir, "finalDir", finalDir)

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
	if err := w.fs.Rename(ctx, tmpDir, finalDir); err != nil {
		_ = w.fs.RemoveAll(tmpDir)
		return "", fmt.Errorf("finalizing snapshot: %w", err)
	}

	return finalDir, nil
}

// copyArtifact copies one file into the snapshot directory.
func (w *Worker) copyArtifact(ctx context.Context, a snapshot.Artifact, srcDir string, dstDir string) error {
	src := filepath.Join(srcDir, a.Name)
	dst := filepath.Join(dstDir, a.Name)
	w.log.Debug("copying artifact", "artifact", a.Name, "src", src, "dst", dst)
	if err := w.fs.CopyFile(ctx, src, dst); err != nil {
		return fmt.Errorf("copying %s: %w", a.Name, err)
	}
	return nil
}

// updateRetentionRules adds to the retention rules the snapshot one
func (w *Worker) updateRetentionRules() {
	w.log.Debug("entering Worker.updateRetentionRules")
	w.mu.RLock()
	mainRule := config.RetentionRule{
		Name:  w.dest.SnapshotSubdir,
		Cron:  "",
		Count: w.dest.Retention.LastCount,
	}

	updated := append([]config.RetentionRule{mainRule}, w.dest.Retention.Rules...)
	w.retention.UpdateConfig(updated)
	w.mu.RUnlock()
}
