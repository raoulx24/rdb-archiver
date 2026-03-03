package fs

import (
	"context"
	"os"
)

// wraps os.Rename with retry logic.
// It provides a resilient, atomic rename operation for snapshotwatcher finalization.

func renameWithRetry(ctx context.Context, cfg Config, oldPath, newPath string) error {
	return retry(ctx, cfg, Operation{Name: "rename"}, func() error {
		return os.Rename(oldPath, newPath)
	})
}
