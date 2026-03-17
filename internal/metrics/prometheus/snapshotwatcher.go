package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/raoulx24/rdb-archiver/internal/snapshotwatcher"
)

type SnapshotWatcherMetrics struct {
	eventsReceived   prometheus.Counter
	snapshotsParsed  prometheus.Counter
	invalidSnapshots prometheus.Counter
	jobsEnqueued     prometheus.Counter
}

func NewSnapshotWatcherMetrics(reg prometheus.Registerer) snapshotwatcher.Metrics {
	m := &SnapshotWatcherMetrics{
		eventsReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_watcher_events_received_total",
			Help: "Number of filesystem events received",
		}),
		snapshotsParsed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_watcher_snapshots_parsed_total",
			Help: "Number of valid snapshots detected",
		}),
		invalidSnapshots: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_watcher_invalid_snapshots_total",
			Help: "Number of invalid or unreadable snapshots",
		}),
		jobsEnqueued: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_watcher_jobs_enqueued_total",
			Help: "Number of jobs enqueued into mailbox",
		}),
	}

	reg.MustRegister(
		m.eventsReceived,
		m.snapshotsParsed,
		m.invalidSnapshots,
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

func (m *SnapshotWatcherMetrics) JobEnqueued() {
	m.jobsEnqueued.Inc()
}
