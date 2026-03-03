package fs

import (
	"context"
	"os"
	"sync"
)

type OSFS struct {
	cfg Config
	mu  sync.RWMutex
}

// the concrete implementation of FS backed by the local OS filesystem.
// Platform-specific details (such as inode extraction) are handled in build-tagged files.

func New(config Config) *OSFS {
	return &OSFS{cfg: config}
}

func (o *OSFS) Stat(path string) (FileInfo, error) {
	st, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}

	return FileInfo{
		Path:  path,
		Size:  st.Size(),
		MTime: st.ModTime(),
		Inode: inodeOf(st),
	}, nil
}

func (o *OSFS) MkdirAll(path string) error { return os.MkdirAll(path, 0o755) }

func (o *OSFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (o *OSFS) CopyFile(ctx context.Context, src, dst string) error {
	o.mu.RLock()
	cfg := o.cfg
	o.mu.RUnlock()
	return copyWithRetry(ctx, o, cfg, src, dst)
}

func (o *OSFS) Rename(ctx context.Context, oldPath, newPath string) error {
	o.mu.RLock()
	cfg := o.cfg
	o.mu.RUnlock()
	return renameWithRetry(ctx, cfg, oldPath, newPath)
}

func (o *OSFS) ReadDir(path string) ([]os.DirEntry, error) { return os.ReadDir(path) }

func (o *OSFS) CopyDir(ctx context.Context, src, dst string) error {
	o.mu.RLock()
	cfg := o.cfg
	o.mu.RUnlock()
	return copyDirWithRetry(ctx, o, cfg, src, dst)
}

func (o *OSFS) UpdateConfig(cfg Config) {
	o.mu.Lock()
	o.cfg = cfg
	o.mu.Unlock()
}
