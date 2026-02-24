//go:build unix

package fs

import (
	"os"
	"syscall"
)

// inode_unix.go extracts inode information from syscall.Stat_t on Unix systems.
// Inode values are used to detect whether the source file changed during copy.

func inodeOf(info os.FileInfo) uint64 {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return st.Ino
}
