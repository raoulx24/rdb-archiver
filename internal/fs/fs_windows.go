//go:build windows

package fs

import (
	"os"

	"golang.org/x/sys/windows"
)

// inodeOf provides a Windows stub for inode extraction.
// Windows does not expose POSIX inodes, so this implementation returns zero.
func inodeOf(info os.FileInfo) uint64 {
	// Windows doesn't expose POSIX inodes in the same way.
	// For our purposes (dev on Windows, run on Linux), 0 is fine.
	_ = info
	return 0
}

// StatFS returns total, free space in path
func (o *OSFS) StatFS(path string) (total uint64, free uint64, err error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}

	var freeBytesAvailable uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64

	err = windows.GetDiskFreeSpaceEx(
		p,
		&freeBytesAvailable,
		&totalNumberOfBytes,
		&totalNumberOfFreeBytes,
	)
	if err != nil {
		return 0, 0, err
	}

	total = totalNumberOfBytes
	free = freeBytesAvailable

	return total, free, nil
}
