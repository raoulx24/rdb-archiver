package worker

import (
	"time"
)

// Job represents a snapshot job submitted to the worker.
type Job struct {
	SourcePath string
	Timestamp  time.Time
}
