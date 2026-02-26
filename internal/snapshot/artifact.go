package snapshot

import (
	"os"
	"path/filepath"
	"time"
)

// Artifact describes a single file within a snapshot
type Artifact struct {
	Name    string
	Size    int64
	ModTime time.Time
}

// FromFileInfo constructs an Artifact from a file path and os.FileInfo.
func FromFileInfo(path string, info os.FileInfo) Artifact {
	return Artifact{
		Name:    filepath.Base(path),
		ModTime: info.ModTime(),
		Size:    info.Size(),
	}
}
