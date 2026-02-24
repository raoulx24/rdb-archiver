package snapshot

import "time"

// Snapshot represents a single archived snapshot.
type Snapshot struct {
	Path      string
	Timestamp time.Time
	Size      int64
}
