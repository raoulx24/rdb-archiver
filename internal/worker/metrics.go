package worker

import "time"

type Metrics interface {
	JobProcessed()
	JobFailed()
	JobRetried()
	ObserveJobProcessingDuration(d time.Duration)
	AddBytesWritten(n int64)
	SetDestinationTotalBytes(n uint64)
	SetDestinationFreeBytes(n uint64)
}

func (w *Worker) RebuildMetrics(newMetrics Metrics) {
	w.logg.Debug("rebuilding metrics", "function", "RebuildMetrics")
	w.mu.Lock()
	defer w.mu.Unlock()

	w.metrics = newMetrics
}
