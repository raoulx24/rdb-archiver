package prometheus

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raoulx24/rdb-archiver/internal/snapshotwatcher"
)

type SnapshotWatcherMetrics struct {
	eventsReceived    prometheus.Counter
	snapshotsParsed   prometheus.Counter
	invalidSnapshots  prometheus.Counter
	detectionDuration prometheus.Histogram
	jobsEnqueued      prometheus.Counter
}

func NewSnapshotWatcherMetrics(reg prometheus.Registerer) snapshotwatcher.Metrics {
	m := &SnapshotWatcherMetrics{
		eventsReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "snapshotwatcher_events_received_total",
			Help: "Number of filesystem events received",
		}),
		snapshotsParsed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "snapshotwatcher_snapshots_parsed_total",
			Help: "Number of valid snapshots detected",
		}),
		invalidSnapshots: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "snapshotwatcher_invalid_snapshots_total",
			Help: "Number of invalid or unreadable snapshots",
		}),
		detectionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "snapshotwatcher_detection_duration_seconds",
			Help:    "Time spent detecting and parsing snapshots",
			Buckets: prometheus.DefBuckets,
		}),
		jobsEnqueued: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "snapshotwatcher_jobs_enqueued_total",
			Help: "Number of jobs enqueued into mailbox",
		}),
	}

	reg.MustRegister(
		m.eventsReceived,
		m.snapshotsParsed,
		m.invalidSnapshots,
		m.detectionDuration,
		m.jobsEnqueued,
	)

	return m
}

func (m *SnapshotWatcherMetrics) EventReceived() {
	m.eventsReceived.Inc()
}

func (m *SnapshotWatcherMetrics) SnapshotParsed() {
	m.snapshotsParsed.Inc()
}

func (m *SnapshotWatcherMetrics) InvalidSnapshot() {
	m.invalidSnapshots.Inc()
}

func (m *SnapshotWatcherMetrics) ObserveDetectionDuration(d time.Duration) {
	m.detectionDuration.Observe(d.Seconds())
}

func (m *SnapshotWatcherMetrics) JobEnqueued() {
	m.jobsEnqueued.Inc()
}
