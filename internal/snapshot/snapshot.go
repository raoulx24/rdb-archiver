// Package snapshot defines the snapshot domain model used by retention logic.
package snapshot

// Snapshot represents a single archived snapshot file.
type Snapshot struct {
	Dir     string
	Primary Artifact
	Aux     []Artifact
}
