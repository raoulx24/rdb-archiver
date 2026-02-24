// Package snapshot defines the snapshot domain model used by retention logic.
package snapshot

import (
	"os"
	"time"
)

// Snapshot represents a single archived snapshot file.
type Snapshot struct {
	Path      string
	Timestamp time.Time
	Size      int64
}

// FromFileInfo constructs a Snapshot from a file path and os.FileInfo.
func FromFileInfo(path string, info os.FileInfo) Snapshot {
	return Snapshot{
		Path:      path,
		Timestamp: info.ModTime(),
		Size:      info.Size(),
	}
}
