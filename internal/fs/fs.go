// Package fs defines the filesystem abstraction used by rdb-archiver.
// It provides the FS interface and the FileInfo type shared across the system.
package fs

import (
	"context"
	"time"
)

type FileInfo struct {
	Path  string
	Size  int64
	MTime time.Time
	Inode uint64
}

type FS interface {
	Stat(path string) (FileInfo, error)
	CopyFile(ctx context.Context, src, dst string) error
	Rename(ctx context.Context, oldPath, newPath string) error
	MkdirAll(path string) error
	RemoveAll(path string) error
}
