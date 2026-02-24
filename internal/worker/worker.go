// Package worker implements the snapshot worker responsible for copying RDB files
// into the archive directory, performing atomic renames, and triggering retention.
package worker

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/raoulx24/rdb-archiver/internal/fs"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/retention"
)

type Worker struct {
	archiveDir string
	log        logging.Logger
	retention  *retention.Engine
	fs         fs.FS
}

func New(archiveDir string, log logging.Logger, r *retention.Engine) *Worker {
	return &Worker{
		archiveDir: archiveDir,
		log:        log,
		retention:  r,
	}
}

// Handle runs the snapshot job and then applies retention.
func (w *Worker) Handle(ctx context.Context, srcPath string) error {
	finalPath, err := w.Run(ctx, srcPath)
	if err != nil {
		return err
	}

	// Trigger retention AFTER the snapshot is finalized
	if err := w.retention.Apply(ctx, w.archiveDir, finalPath); err != nil {
		w.log.Error("worker: retention failed: %v", err)
	}

	return nil
}

// Run executes a single snapshot job and returns the final archived file path.
func (w *Worker) Run(ctx context.Context, srcPath string) (string, error) {
	tmpPath := filepath.Join(w.archiveDir, ".tmp-"+filepath.Base(srcPath))
	finalPath := filepath.Join(w.archiveDir, filepath.Base(srcPath))

	if err := w.fs.MkdirAll(w.archiveDir); err != nil {
		return "", fmt.Errorf("creating archive dir: %w", err)
	}

	if err := w.fs.CopyFile(ctx, srcPath, tmpPath); err != nil {
		return "", fmt.Errorf("copying snapshot: %w", err)
	}

	if err := w.fs.Rename(ctx, tmpPath, finalPath); err != nil {
		return "", fmt.Errorf("finalizing snapshot: %w", err)
	}

	return finalPath, nil
}
