// Package worker processes snapshotwatcher jobs and writes atomic snapshotwatcher directories.
package worker

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

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
	metrics   Metrics
	fs        fs.FS
	logg      logging.Logger
	retention *retention.Retention
	mb        *mailbox.Mailbox[snapshot.Job]
}

// New creates a worker using destination config and mailbox.
func New(cfg Config, log logging.Logger, metrics Metrics, r *retention.Retention, mb *mailbox.Mailbox[snapshot.Job], filesystem fs.FS) *Worker {
	logg := log.With("pkg", "worker")
	logg.Debug("creating worker")
	return &Worker{
		cfg:       cfg,
		metrics:   metrics,
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
	w.updateRetentionRules(w.cfg)
	for {
		job, ok := w.mb.Take(ctx)
		if !ok {
			w.logg.Info("worker stopped")
			return
		}
		start := time.Now()

		if err := w.Handle(ctx, job.Snap); err != nil {
			w.metrics.JobFailed()
			w.logg.Error("snapshot handle failed", "error", err)
		} else {
			w.metrics.JobProcessed()
		}

		w.metrics.ObserveJobProcessingDuration(time.Since(start))
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
	if !isSameConfig(cfg, w.cfg) {
		w.cfg = cfg
		w.logg.Info("config updated")
	}
	w.mu.Unlock()

	w.updateRetentionRules(cfg)
}

// writeSnapshot creates a tar+compressed archive for all snapshot files atomically.
func (w *Worker) writeSnapshot(ctx context.Context, snap snapshot.Snapshot) (string, error) {
	w.mu.RLock()
	dest := w.cfg
	w.mu.RUnlock()

	root := filepath.Join(dest.Root, dest.SubDir)
	snapDir := filepath.Join(root, dest.SnapshotSubdir)

	ts := snap.Primary.ModTime.UTC().Format("2006-01-02T15-04-05")

	// For now we fix the extension to .tar.zst; algorithm/level are hidden in fs.Config.
	tmpArchive := filepath.Join(snapDir, ".tmp-"+ts+".tar.zst")
	finalArchive := filepath.Join(snapDir, ts+".tar.zst")

	w.logg.Debug("new destinations", "tmpArchive", tmpArchive, "finalArchive", finalArchive)

	if err := w.fs.MkdirAll(snapDir); err != nil {
		return "", fmt.Errorf("creating snapshot dir: %w", err)
	}

	// Collect all artifact names (primary + aux) relative to snap.Dir.
	files := make([]string, 0, 1+len(snap.Aux))
	files = append(files, snap.Primary.Name)
	for _, a := range snap.Aux {
		files = append(files, a.Name)
	}

	// Create compressed tar archive into tmp file.
	if err := w.fs.CreateCompressedTar(ctx, snap.Dir, files, tmpArchive); err != nil {
		_ = w.fs.RemoveAll(tmpArchive)
		return "", fmt.Errorf("creating compressed archive: %w", err)
	}

	// Get file size
	if info, err := w.fs.Stat(tmpArchive); err == nil {
		w.metrics.AddBytesWritten(info.Size)
	}

	// Finalize atomically: remove existing final archive if present, then rename.
	if _, err := w.fs.Stat(finalArchive); err == nil {
		if err := w.fs.RemoveAll(finalArchive); err != nil {
			return "", fmt.Errorf("failed to remove existing final archive: %w", err)
		}
	}

	if err := w.fs.Rename(ctx, tmpArchive, finalArchive); err != nil {
		_ = w.fs.RemoveAll(tmpArchive)
		return "", fmt.Errorf("finalizing snapshot archive: %w", err)
	}

	return finalArchive, nil
}

// updateRetentionRules adds to the retention rules the snapshotwatcher one
func (w *Worker) updateRetentionRules(cfg Config) {
	w.logg.Debug("entering Worker.updateRetentionRules")
	w.mu.RLock()
	mainRule := retention.Rule{
		Name:  cfg.SnapshotSubdir,
		Cron:  "",
		Count: cfg.Retention.LastCount,
	}
	updated := append([]retention.Rule{mainRule}, cfg.Retention.Rules...)
	removeUnknownFolders := cfg.Retention.RemoveUnknownFolders
	w.mu.RUnlock()
	w.retention.UpdateConfig(retention.Config{RemoveUnknownFolders: removeUnknownFolders, Rules: updated})
}
