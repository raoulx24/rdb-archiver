package prometheus

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raoulx24/rdb-archiver/internal/worker"
)

type WorkerMetrics struct {
	jobsProcessed prometheus.Counter
	jobsFailed    prometheus.Counter
	jobsRetried   prometheus.Counter

	jobDuration  prometheus.Histogram
	bytesWritten prometheus.Counter
}

func NewWorkerMetrics(reg prometheus.Registerer, cfg Config) worker.Metrics {
	m := &WorkerMetrics{
		jobsProcessed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_jobs_processed_total",
			Help: "Number of jobs successfully processed",
		}),
		jobsFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_jobs_failed_total",
			Help: "Number of jobs that failed",
		}),
		jobsRetried: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_jobs_retried_total",
			Help: "Number of jobs retried",
		}),
		jobDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "rdb_archiver_job_processing_duration_seconds",
			Help:    "Time spent processing a job",
			Buckets: cfg.HistogramBuckets,
		}),
		bytesWritten: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_bytes_written_total",
			Help: "Total bytes written by worker",
		}),
	}

	reg.MustRegister(
		m.jobsProcessed,
		m.jobsFailed,
		m.jobsRetried,
		m.jobDuration,
		m.bytesWritten,
	)

	return m
}

func (m *WorkerMetrics) JobProcessed() {
	m.jobsProcessed.Inc()
}

func (m *WorkerMetrics) JobFailed() {
	m.jobsFailed.Inc()
}

func (m *WorkerMetrics) JobRetried() {
	m.jobsRetried.Inc()
}

func (m *WorkerMetrics) ObserveJobProcessingDuration(d time.Duration) {
	m.jobDuration.Observe(d.Seconds())
}

func (m *WorkerMetrics) AddBytesWritten(n int64) {
	m.bytesWritten.Add(float64(n))
}
