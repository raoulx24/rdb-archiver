package fs

import (
	"context"
	"os"
)

// wraps os.Rename with retry logic.
// It provides a resilient, atomic rename operation for snapshot finalization.

func renameWithRetry(ctx context.Context, oldPath, newPath string) error {
	return retry(ctx, "rename", func() error {
		return os.Rename(oldPath, newPath)
	})
}
