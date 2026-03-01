package snapshotwatcher

import (
	"os"
	"path/filepath"
	"time"
)

// Snapshot represents a single archived snapshotwatcher file.
type Snapshot struct {
	Dir     string
	Primary Artifact
	Aux     []Artifact
}

// Job wraps a snapshotwatcher for mailbox delivery.
type Job struct {
	Snap Snapshot
}

// Artifact describes a single file within a snapshotwatcher
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
