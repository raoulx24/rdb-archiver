package fs

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
)

// createCompressedTarWithRetry creates a tar+compressed archive of the given files
// (relative to srcDir) into dst, with retry and source-change detection.
func createCompressedTarWithRetry(ctx context.Context, f FS, cfg Config, srcDir string, files []string, dst string) error {
	// Capture original metadata for all files.
	orig := make(map[string]FileInfo, len(files))
	for _, name := range files {
		full := filepath.Join(srcDir, name)
		fi, err := f.Stat(full)
		if err != nil {
			return fmt.Errorf("stat %s: %w", full, err)
		}
		orig[name] = fi
	}

	op := Operation{Name: "compress-tar"}

	return retry(ctx, cfg, op, func() error {
		// Re-check all files before each attempt.
		for _, name := range files {
			full := filepath.Join(srcDir, name)
			now, err := f.Stat(full)
			if err != nil {
				return fmt.Errorf("stat %s: %w", full, err)
			}
			if sourceChanged(orig[name], now) {
				return fmt.Errorf("source changed during compression: %s", full)
			}
		}

		return createCompressedTarOnce(srcDir, files, dst, cfg.CompressionLevel)
	})
}

func createCompressedTarOnce(srcDir string, files []string, dst string, level int) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	// zstd encoder with configurable level.
	if level <= 0 {
		level = 2 // sane default if not set
	}
	enc, err := zstd.NewWriter(out, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)))
	if err != nil {
		return fmt.Errorf("creating zstd writer: %w", err)
	}
	defer enc.Close()

	tw := tar.NewWriter(enc)
	defer tw.Close()

	for _, name := range files {
		full := filepath.Join(srcDir, name)

		st, err := os.Stat(full)
		if err != nil {
			return fmt.Errorf("stat %s: %w", full, err)
		}

		hdr, err := tar.FileInfoHeader(st, "")
		if err != nil {
			return fmt.Errorf("tar header %s: %w", full, err)
		}
		// Preserve relative path inside archive.
		hdr.Name = name

		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("tar write header %s: %w", full, err)
		}

		in, err := os.Open(full)
		if err != nil {
			return fmt.Errorf("open %s: %w", full, err)
		}

		if _, err := io.Copy(tw, in); err != nil {
			_ = in.Close()
			return fmt.Errorf("copy %s: %w", full, err)
		}
		_ = in.Close()
	}

	// Flush tar + zstd + file.
	if err := tw.Close(); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return out.Sync()
}
