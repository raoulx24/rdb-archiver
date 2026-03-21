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

	jobDuration prometheus.Histogram

	bytesWritten prometheus.Counter

	destinationTotalBytes prometheus.Gauge
	destinationFreeBytes  prometheus.Gauge
}

func NewWorkerMetrics(reg prometheus.Registerer, cfg Config) worker.Metrics {
	m := &WorkerMetrics{
		jobsProcessed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_worker_jobs_processed_total",
			Help: "Number of jobs successfully processed",
		}),
		jobsFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_worker_jobs_failed_total",
			Help: "Number of jobs that failed",
		}),
		jobsRetried: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_worker_jobs_retried_total",
			Help: "Number of jobs retried",
		}),
		jobDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "rdb_archiver_worker_job_processing_duration_seconds",
			Help:    "Time spent processing a job",
			Buckets: cfg.HistogramBuckets,
		}),
		bytesWritten: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_worker_bytes_written_total",
			Help: "Total bytes written by worker",
		}),
		destinationTotalBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rdb_archiver_worker_destination_total_bytes",
			Help: "Total bytes of the destination filesystem",
		}),
		destinationFreeBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rdb_archiver_worker_destination_free_bytes",
			Help: "Free bytes available on the destination filesystem",
		}),
	}

	reg.MustRegister(
		m.jobsProcessed,
		m.jobsFailed,
		m.jobsRetried,
		m.jobDuration,
		m.bytesWritten,
		m.destinationTotalBytes,
		m.destinationFreeBytes,
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

func (m *WorkerMetrics) SetDestinationTotalBytes(n uint64) {
	m.destinationTotalBytes.Set(float64(n))
}

func (m *WorkerMetrics) SetDestinationFreeBytes(n uint64) {
	m.destinationFreeBytes.Set(float64(n))
}
