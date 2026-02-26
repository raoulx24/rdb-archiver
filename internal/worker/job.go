package worker

import "github.com/raoulx24/rdb-archiver/internal/snapshot"

// Job wraps a snapshot for mailbox delivery.
type Job struct {
	Snap snapshot.Snapshot
}
