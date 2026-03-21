//go:build unix

package fs

import (
	"os"
	
	"golang.org/x/sys/unix"
)

// inodeOf extracts inode information from syscall.Stat_t on Unix systems.
// Inode values are used to detect whether the source file changed during copy.
func inodeOf(info os.FileInfo) uint64 {
	st, ok := info.Sys().(*unix.Stat_t)
	if !ok {
		return 0
	}
	return st.Ino
}

// StatFS returns total, free space in path, at OS level
func (o *OSFS) StatFS(path string) (total uint64, free uint64, err error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}

	// Total bytes in filesystem
	total = stat.Blocks * uint64(stat.Bsize)

	// Free bytes available to unprivileged processes (matches df -h)
	free = stat.Bavail * uint64(stat.Bsize)

	return total, free, nil
}
