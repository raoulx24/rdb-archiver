package snapshotwatcher

import (
	"os"
	"path/filepath"

	"github.com/raoulx24/rdb-archiver/internal/snapshot"
)

// loadAux loads auxiliary artifacts if present.
func (sw *Watcher) loadAux(dir string, names []string) []snapshot.Artifact {
	var out []snapshot.Artifact

	for _, name := range names {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			sw.logg.Error("cannot stat aux file", "error", err, "dir", dir, "name", name)
			continue
		}
		sw.logg.Debug("adding aux file to snapshot", "function", "loadAux", "dir", dir, "name", name)
		out = append(out, snapshot.FromFileInfo(path, info))
	}

	return out
}
