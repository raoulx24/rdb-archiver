package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
)

// loadAux loads auxiliary files associated with the primary file.
func (w *Watcher) loadAux(dir string, auxNames []string) []Artifact {
	var auxArtifacts []Artifact

	for _, auxName := range auxNames {
		auxPath := filepath.Join(dir, auxName)

		info, err := os.Stat(auxPath)
		if err != nil {
			fmt.Printf("Error loading aux file %s: %v\n", auxPath, err)
			continue
		}

		auxArtifact := FromFileInfo(auxPath, info)
		auxArtifacts = append(auxArtifacts, auxArtifact)
	}

	return auxArtifacts
}
