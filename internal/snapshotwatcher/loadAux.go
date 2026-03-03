package snapshotwatcher

import (
	"os"
	"path/filepath"

	"github.com/raoulx24/rdb-archiver/internal/snapshot"
)

// loadAux loads auxiliary artifacts if present.
func (sw *SnapshotWatcher) loadAux(dir string, names []string) []snapshot.Artifact {
	var out []snapshot.Artifact

	for _, name := range names {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		out = append(out, snapshot.FromFileInfo(path, info))
	}

	return out
}
