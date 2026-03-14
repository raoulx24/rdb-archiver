package retention

import "time"

type Metrics interface {
	RetentionRun()
	ObserveRetentionRunDuration(d time.Duration)

	SnapshotProcessed(rule string, outcome string)
	SnapshotDeleted(rule string)
}
