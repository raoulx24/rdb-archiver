package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/raoulx24/rdb-archiver/internal/retention"
)

type RetentionMetrics struct {
	retentionRuns      prometheus.Counter
	snapshotsProcessed *prometheus.CounterVec
	snapshotsDeleted   *prometheus.CounterVec
}

func NewRetentionMetrics(reg prometheus.Registerer) retention.Metrics {
	m := &RetentionMetrics{
		retentionRuns: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_retention_runs_total",
			Help: "Number of retention cycles executed",
		}),

		snapshotsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rdb_archiver_retention_snapshots_processed_total",
				Help: "Snapshots processed per rule and outcome",
			},
			[]string{"rule", "outcome"},
		),

		snapshotsDeleted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rdb_archiver_retention_deletions_total",
				Help: "Snapshots deleted per rule",
			},
			[]string{"rule"},
		),
	}

	reg.MustRegister(
		m.retentionRuns,
		m.snapshotsProcessed,
		m.snapshotsDeleted,
	)

	return m
}

func (m *RetentionMetrics) RetentionRun() {
	m.retentionRuns.Inc()
}

func (m *RetentionMetrics) SnapshotProcessed(rule string, outcome string) {
	m.snapshotsProcessed.WithLabelValues(rule, outcome).Inc()
}

func (m *RetentionMetrics) SnapshotDeleted(rule string) {
	m.snapshotsDeleted.WithLabelValues(rule).Inc()
}
