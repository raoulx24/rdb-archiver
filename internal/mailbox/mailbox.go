package mailbox

import (
	"context"
	"sync"

	"github.com/raoulx24/rdb-archiver/internal/logging"
)

// Mailbox is a single-slot buffer where the latest job always wins.
// It is NOT a queue. It holds at most one pending job.
// Put() overwrites any existing job. Take() blocks until a job is available.
type Mailbox[T any] struct {
	mu      sync.Mutex
	logg    logging.Logger
	metrics Metrics
	cond    *sync.Cond
	job     *T
	stopped bool
}

// New creates an empty mailbox.
func New[T any](metrics Metrics, log logging.Logger) *Mailbox[T] {
	logg := log.With("pkg", "mailbox")
	logg.Debug("creating mailbox", "function", "New")
	m := &Mailbox[T]{
		logg:    logg,
		metrics: metrics,
	}
	m.cond = sync.NewCond(&m.mu)

	return m
}

// Stop stops the mailbox's background goroutine.
func (m *Mailbox[T]) Stop() {
	m.logg.Debug("stopping mailbox", "function", "Stop")
	m.mu.Lock()
	m.stopped = true
	m.cond.Broadcast()
	m.mu.Unlock()
}

// Put stores a job in the mailbox, replacing any existing job.
// It never blocks.
func (m *Mailbox[T]) Put(j T) {
	m.logg.Debug("new job", "function", "Put")
	m.mu.Lock()
	metrics := m.metrics

	prev := m.job

	m.job = &j

	m.cond.Signal() // wake up worker if waiting - and keep it in lock()
	m.mu.Unlock()

	if prev != nil {
		m.logg.Debug("overriding job", "function", "Put")
		metrics.JobOverwritten()
	}

	metrics.JobEnqueued()
	m.logg.Debug("job enqueued", "function", "Put")
}

// Take blocks until a job is available, then returns it and clears the slot.
func (m *Mailbox[T]) Take(ctx context.Context) (T, bool) {
	m.logg.Debug("dequeueing job", "function", "Take")

	var zero T

	// sync.Cond does not support context cancellation.
	// To allow Take() to unblock when the context is cancelled,
	// we register a callback that broadcasts the condition.
	// This wakes any goroutine currently waiting in cond.Wait().
	stop := context.AfterFunc(ctx, func() {
		m.mu.Lock()
		m.cond.Broadcast()
		m.mu.Unlock()
	})
	defer stop() // Ensure the callback is removed if we return normally.

	m.mu.Lock()
	metrics := m.metrics

	// Wait until:
	//   - a job is available
	//   - the mailbox is stopped
	//   - the context is cancelled
	//
	// cond.Wait() releases the mutex while waiting and re-acquires it
	// before returning.
	for m.job == nil && !m.stopped {
		// If the context was cancelled, stop waiting and return.
		if ctx.Err() != nil {
			m.mu.Unlock()
			return zero, false
		}

		// Wait for:
		//   - Put() to signal a new job
		//   - Stop() to broadcast shutdown
		//   - the context.AfterFunc() callback to broadcast cancellation
		m.cond.Wait()
	}

	// If the mailbox was stopped and there is no job left, exit.
	if m.job == nil {
		m.mu.Unlock()
		return zero, false
	}

	// Retrieve the job and clear the mailbox slot.
	j := *m.job
	m.job = nil

	m.mu.Unlock()

	// Update metrics outside the lock to avoid blocking mailbox operations.
	metrics.JobDequeued()

	m.logg.Debug("job dequeued", "function", "Take")
	return j, true
}

// TryTake returns the job if present, or nil if empty.
// It never blocks.
func (m *Mailbox[T]) TryTake() *T {
	m.logg.Debug("try dequeueing job", "function", "TryTake")
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := m.metrics

	if m.job == nil {
		return nil
	}

	j := m.job
	m.job = nil

	metrics.JobDequeued()
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
