package watcher

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/raoulx24/rdb-archiver/internal/snapshot"
)

// loadAux loads auxiliary files associated with the primary file.
func (w *Watcher) loadAux(dir string, auxNames []string) []snapshot.Artifact {
	var auxArtifacts []snapshot.Artifact

	for _, auxName := range auxNames {
		auxPath := filepath.Join(dir, auxName)

		info, err := os.Stat(auxPath)
		if err != nil {
			fmt.Printf("Error loading aux file %s: %v\n", auxPath, err)
			continue
		}

		auxArtifact := snapshot.FromFileInfo(auxPath, info)
		auxArtifacts = append(auxArtifacts, auxArtifact)
	}

	return auxArtifacts
}
