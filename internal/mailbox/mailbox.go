package mailbox

import (
	"context"
	"sync"
	"time"
)

// Mailbox is a single-slot buffer where the latest job always wins.
// It is NOT a queue. It holds at most one pending job.
// Put() overwrites any existing job. Take() blocks until a job is available.
type Mailbox[T any] struct {
	mu           sync.Mutex
	metrics      Metrics
	cond         *sync.Cond
	job          *T
	jobTimestamp time.Time
	stopCh       chan struct{}
}

// New creates an empty mailbox.
func New[T any](metrics Metrics) *Mailbox[T] {
	m := &Mailbox[T]{
		metrics: metrics,
		stopCh:  make(chan struct{}),
	}
	m.cond = sync.NewCond(&m.mu)

	go m.ageUpdater()

	return m
}

// Stop stops the mailbox's background goroutine.
func (m *Mailbox[T]) Stop() {
	close(m.stopCh)
}

// Put stores a job in the mailbox, replacing any existing job.
// It never blocks.
func (m *Mailbox[T]) Put(j T) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.job != nil {
		m.metrics.JobOverwritten()
	}

	m.job = &j

	m.jobTimestamp = time.Now()
	m.metrics.JobEnqueued()
	m.metrics.SetCurrentJobAge(0)

	m.cond.Signal() // wake up worker if waiting
}

// Take blocks until a job is available, then returns it and clears the slot.
func (m *Mailbox[T]) Take(ctx context.Context) (T, bool) {
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

	return j, true
}

// TryTake returns the job if present, or nil if empty.
// It never blocks.
func (m *Mailbox[T]) TryTake() *T {
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

	return j
}

// HasJob reports whether a job is currently waiting.
func (m *Mailbox[T]) HasJob() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.job != nil
}

// ageUpdater runs in the background and updates job age every second.
func (m *Mailbox[T]) ageUpdater() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			if m.job != nil && !m.jobTimestamp.IsZero() {
				age := time.Since(m.jobTimestamp).Seconds()
				m.metrics.SetCurrentJobAge(age)
			}
			m.mu.Unlock()

		case <-m.stopCh:
			return
		}
	}
}
