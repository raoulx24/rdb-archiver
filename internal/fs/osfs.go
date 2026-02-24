package fs

import (
	"context"
	"os"
)

type OSFS struct{}

// the concrete implementation of FS backed by the local OS filesystem.
// Platform-specific details (such as inode extraction) are handled in build-tagged files.

func New() *OSFS {
	return &OSFS{}
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

func (o *OSFS) MkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (o *OSFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (o *OSFS) CopyFile(ctx context.Context, src, dst string) error {
	return copyWithRetry(ctx, o, src, dst)
}

func (o *OSFS) Rename(ctx context.Context, oldPath, newPath string) error {
	return renameWithRetry(ctx, oldPath, newPath)
}
