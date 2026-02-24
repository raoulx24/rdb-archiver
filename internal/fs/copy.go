package fs

import (
	"context"
	"fmt"
	"io"
	"os"
)

// implements file copying with retry and source-change detection.
// It ensures that snapshot copies are consistent and aborts if the source file changes mid-copy.

func copyWithRetry(ctx context.Context, f FS, src, dst string) error {
	orig, err := f.Stat(src)
	if err != nil {
		return err
	}

	return retry(ctx, "copy", func() error {
		now, err := f.Stat(src)
		if err != nil {
			return err
		}

		if sourceChanged(orig, now) {
			return fmt.Errorf("source changed during copy")
		}

		return copyOnce(src, dst)
	})
}

func sourceChanged(orig, now FileInfo) bool {
	if now.Inode != 0 && orig.Inode != 0 && now.Inode != orig.Inode {
		return true
	}
	if now.MTime.After(orig.MTime) {
		return true
	}
	if now.Size != orig.Size {
		return true
	}
	return false
}

func copyOnce(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}
