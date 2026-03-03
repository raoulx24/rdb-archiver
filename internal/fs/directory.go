package fs

import (
	"context"
	"path/filepath"
)

// copyDirWithRetry copies a snapshotwatcher directory recursively.
func copyDirWithRetry(ctx context.Context, f FS, cfg Config, src, dst string) error {
	if err := f.MkdirAll(dst); err != nil {
		return err
	}

	entries, err := f.ReadDir(src)
	if err != nil {
		return err
	}

	for _, ent := range entries {
		s := filepath.Join(src, ent.Name())
		d := filepath.Join(dst, ent.Name())

		if ent.IsDir() {
			if err := copyDirWithRetry(ctx, f, cfg, s, d); err != nil {
				return err
			}
			continue
		}

		if err := copyWithRetry(ctx, f, cfg, s, d); err != nil {
			return err
		}
	}

	return nil
}
