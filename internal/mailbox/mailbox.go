package mailbox

import (
	"context"
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/logging"
)

// Mailbox is a single-slot buffer where the latest job always wins.
// It is NOT a queue. It holds at most one pending job.
// Put() overwrites any existing job. Take() blocks until a job is available.
type Mailbox[T any] struct {
	mu           sync.Mutex
	logg         logging.Logger
	metrics      Metrics
	cond         *sync.Cond
	job          *T
	jobTimestamp time.Time
	stopCh       chan struct{}
}

// New creates an empty mailbox.
func New[T any](metrics Metrics, log logging.Logger) *Mailbox[T] {
	logg := log.With("pkg", "mailbox")
	logg.Debug("creating mailbox", "function", "New")
	m := &Mailbox[T]{
		logg:    logg,
		metrics: metrics,
		stopCh:  make(chan struct{}),
	}
	m.cond = sync.NewCond(&m.mu)

	go m.ageUpdater()

	return m
}

// Stop stops the mailbox's background goroutine.
func (m *Mailbox[T]) Stop() {
	m.logg.Debug("stopping mailbox", "function", "Stop")
	close(m.stopCh)
}

// Put stores a job in the mailbox, replacing any existing job.
// It never blocks.
func (m *Mailbox[T]) Put(j T) {
	m.logg.Debug("new job", "function", "Put")
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.job != nil {
		m.logg.Debug("overriding job", "function", "Put")
		m.metrics.JobOverwritten()
	}

	m.job = &j

	m.jobTimestamp = time.Now()
	m.metrics.JobEnqueued()
	m.metrics.SetCurrentJobAge(0)
	m.logg.Debug("job enqueued", "function", "Put")

	m.cond.Signal() // wake up worker if waiting
}

// Take blocks until a job is available, then returns it and clears the slot.
func (m *Mailbox[T]) Take(ctx context.Context) (T, bool) {
	m.logg.Debug("dequeueing job", "function", "Take")
	var zero T
	m.mu.Lock()
	defer m.mu.Unlock()

	for m.job == nil {
		if ctx.Err() != nil {
			return zero, false
		}
		m.cond.Wait()
	}

	j := *m.job
	m.job = nil
	m.jobTimestamp = time.Time{}

	m.metrics.JobDequeued()
	m.metrics.SetCurrentJobAge(0)
	m.logg.Debug("job dequeued", "function", "Take")

	return j, true
}

// TryTake returns the job if present, or nil if empty.
// It never blocks.
func (m *Mailbox[T]) TryTake() *T {
	m.logg.Debug("dequeueing job", "function", "TryTake")
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.job == nil {
		return nil
	}

	j := m.job
	m.job = nil
	m.jobTimestamp = time.Time{}

	m.metrics.JobDequeued()
	m.metrics.SetCurrentJobAge(0)
	m.logg.Debug("job dequeued", "function", "TryTake")

	return j
}

// HasJob reports whether a job is currently waiting.
func (m *Mailbox[T]) HasJob() bool {
	m.logg.Debug("checking for job", "function", "HasJob")
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.job != nil
}

// ageUpdater runs in the background and updates job age every second.
func (m *Mailbox[T]) ageUpdater() {
	m.logg.Debug("staring age uppdater", "function", "ageUpdater")
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			if m.job != nil && !m.jobTimestamp.IsZero() {
				age := time.Since(m.jobTimestamp).Seconds()
				m.metrics.SetCurrentJobAge(age)
				m.logg.Debug("age updated", "function", "ageUpdater")
			}
			m.mu.Unlock()

		case <-m.stopCh:
			return
		}
	}
}
