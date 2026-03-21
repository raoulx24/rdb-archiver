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

const snapshotExtension = ".tar.zst"

// New creates a worker using destination config and mailbox.
func New(cfg Config, log logging.Logger, metrics Metrics, r *retention.Retention, mb *mailbox.Mailbox[snapshot.Job], filesystem fs.FS) *Worker {
	logg := log.With("pkg", "worker")
	logg.Debug("creating worker", "function", "New")
	return &Worker{
		cfg:       cfg,
		metrics:   metrics,
		fs:        filesystem,
		logg:      logg,
		retention: r,
		mb:        mb,
	}
}

// Start runs the worker loop using mailbox semantics.
func (w *Worker) Start(ctx context.Context) {
	w.logg.Info("starting worker")
	w.updateRetentionRules(w.cfg)
	for {
		w.mu.RLock()
		metrics := w.metrics
		w.mu.RUnlock()
		job, ok := w.mb.Take(ctx)
		if !ok {
			w.logg.Info("worker stopped")
			return
		}
		start := time.Now()

		if err := w.Handle(ctx, metrics, job.Snap); err != nil {
			metrics.JobFailed()
			w.logg.Error("snapshot handle failed", "error", err)
		} else {
			metrics.JobProcessed()
		}

		metrics.ObserveJobProcessingDuration(time.Since(start))
	}
}

// Handle writes a snapshotwatcher directory and applies retention.
func (w *Worker) Handle(ctx context.Context, metrics Metrics, snap snapshot.Snapshot) error {
	w.mu.RLock()
	dest := w.cfg
	w.mu.RUnlock()

	root := filepath.Join(dest.Root, dest.SubDir)

	w.logg.Debug("worker starting snapshot handling", "function", "Handle")
	finalDir, err := w.writeSnapshot(ctx, dest, metrics, snap)
	if err != nil {
		w.updateDestinationDirSizeMetrics(root, metrics)
		w.logg.Error("failed to write snapshot", "error", err)
		return err
	}

	w.logg.Debug("destination root resolved", "function", "Handle", "root", root)

	if err := w.retention.Apply(ctx, w.fs, root, finalDir); err != nil {
		w.logg.Error("worker: retention failed", "error", err)
	}

	w.updateDestinationDirSizeMetrics(root, metrics)

	return nil
}

func (w *Worker) UpdateConfig(cfg Config) {
	w.mu.Lock()
	if !isSameConfig(cfg, w.cfg) {
		w.cfg = cfg
		w.logg.Info("config updated")
	}
	w.mu.Unlock()
	w.logg.Debug("same config, returning", "function", "UpdateConfig")

	w.updateRetentionRules(cfg)
}

// writeSnapshot creates a tar+compressed archive for all snapshot files atomically.
func (w *Worker) writeSnapshot(ctx context.Context, cfg Config, metrics Metrics, snap snapshot.Snapshot) (string, error) {
	root := filepath.Join(cfg.Root, cfg.SubDir)
	snapDir := filepath.Join(root, cfg.SnapshotSubdir)

	ts := snap.Primary.ModTime.UTC().Format("2006-01-02T15-04-05")

	// For now we fix the extension to .tar.zst; algorithm/level are hidden in fs.Config.
	tmpArchive := filepath.Join(snapDir, ".tmp-"+ts+snapshotExtension)
	finalArchive := filepath.Join(snapDir, ts+snapshotExtension)

	w.logg.Debug("new destinations", "function", "writeSnapshot", "tmpArchive", tmpArchive, "finalArchive", finalArchive)

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
		w.logg.Debug("wrote snapshot archive", "function", "writeSnapshot", "tmpArchive", tmpArchive, "archiveSize", info.Size)
		metrics.AddBytesWritten(info.Size)
	}

	// Finalize atomically: remove existing final archive if present, then rename.
	if _, err := w.fs.Stat(finalArchive); err == nil {
		w.logg.Debug("found existing final archive, removing", "function", "writeSnapshot", "finalArchive", finalArchive)
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
	w.mu.RLock()
	mainRule := retention.Rule{
		Name:  cfg.SnapshotSubdir,
		Cron:  "",
		Count: cfg.Retention.LastCount,
	}
	updated := append([]retention.Rule{mainRule}, cfg.Retention.Rules...)
	removeUnknownFolders := cfg.Retention.RemoveUnknownFolders
	w.mu.RUnlock()
	ruleNames := []string{}
	ruleCrons := []string{}
	ruleCounts := []int{}
	for _, rule := range updated {
		ruleNames = append(ruleNames, rule.Name)
		ruleCrons = append(ruleCrons, rule.Cron)
		ruleCounts = append(ruleCounts, rule.Count)
	}
	w.logg.Debug("updating retention rules", "function", "updateRetentionRules", "removeUnknownFolders",
		removeUnknownFolders, "ruleNames",
		ruleNames, "ruleCrons", ruleCrons, "ruleCounts", ruleCounts)
	w.retention.UpdateConfig(retention.Config{RemoveUnknownFolders: removeUnknownFolders, SnapshotExtension: snapshotExtension, Rules: updated})
}

func (w *Worker) updateDestinationDirSizeMetrics(path string, metrics Metrics) {
	total, free, err := w.fs.StatFS(path)
	if err == nil {
		metrics.SetDestinationTotalBytes(total)
		metrics.SetDestinationFreeBytes(free)
	} else {
		w.logg.Warn("failed to stat filesystem", "error", err)
	}
}
