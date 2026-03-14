package worker

import "time"

type Metrics interface {
	JobProcessed()
	JobFailed()
	JobRetried()
	ObserveJobProcessingDuration(d time.Duration)
	AddBytesWritten(n int64)
}
