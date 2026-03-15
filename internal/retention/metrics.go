package retention

import "time"

type Metrics interface {
	RetentionRun()
	ObserveRetentionRunDuration(d time.Duration)

	SnapshotProcessed(rule string, outcome string)
	SnapshotDeleted(rule string)
}

func (r *Retention) RebuildMetrics(newMetrics Metrics) {
	r.logg.Debug("rebuilding metrics", "function", "RebuildMetrics")
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics = newMetrics
}
