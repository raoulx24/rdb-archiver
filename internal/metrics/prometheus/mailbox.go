package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/raoulx24/rdb-archiver/internal/mailbox"
)

type MailboxMetrics struct {
	enqueued    prometheus.Counter
	overwritten prometheus.Counter
	dequeued    prometheus.Counter
}

func NewMailboxMetrics(reg prometheus.Registerer) mailbox.Metrics {
	m := &MailboxMetrics{
		enqueued: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_mailbox_jobs_enqueued_total",
			Help: "Number of jobs enqueued into mailbox",
		}),
		overwritten: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_mailbox_jobs_overwritten_total",
			Help: "Number of jobs overwritten in mailbox",
		}),
		dequeued: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rdb_archiver_mailbox_jobs_dequeued_total",
			Help: "Number of jobs dequeued from mailbox",
		}),
	}

	reg.MustRegister(
		m.enqueued,
		m.overwritten,
		m.dequeued,
	)

	return m
}

func (m *MailboxMetrics) JobEnqueued() {
	m.enqueued.Inc()
}

func (m *MailboxMetrics) JobOverwritten() {
	m.overwritten.Inc()
}

func (m *MailboxMetrics) JobDequeued() {
	m.dequeued.Inc()
}
