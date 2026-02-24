//go:build windows

package fs

import "os"

// provides a Windows stub for inode extraction.
// Windows does not expose POSIX inodes, so this implementation returns zero.

func inodeOf(info os.FileInfo) uint64 {
	// Windows doesn't expose POSIX inodes in the same way.
	// For our purposes (dev on Windows, run on Linux), 0 is fine.
	_ = info
	return 0
}
