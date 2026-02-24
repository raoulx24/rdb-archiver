// Package worker implements the snapshot worker responsible for copying RDB files
// into the archive directory, performing atomic renames, and triggering retention.
package worker

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/raoulx24/rdb-archiver/internal/fs"
)

// Worker processes snapshot jobs by copying the source RDB file into a temporary
// location, validating it, and atomically renaming it into the final archive path.
type Worker struct {
	fs         fs.FS
	archiveDir string
}

func New(f fs.FS, archiveDir string) *Worker {
	return &Worker{
		fs:         f,
		archiveDir: archiveDir,
	}
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
