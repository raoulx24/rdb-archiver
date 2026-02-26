package mailbox

import "sync"

// Mailbox is a single-slot buffer where the latest job always wins.
// It is NOT a queue. It holds at most one pending job.
// Put() overwrites any existing job. Take() blocks until a job is available.
type Mailbox[T any] struct {
	mu   sync.Mutex
	cond *sync.Cond
	job  *T
}

// New creates an empty mailbox.
func New[T any]() *Mailbox[T] {
	m := &Mailbox[T]{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

// Put stores a job in the mailbox, replacing any existing job.
// It never blocks.
func (m *Mailbox[T]) Put(j T) {
	m.mu.Lock()
	m.job = &j
	m.mu.Unlock()
	m.cond.Signal() // wake up worker if waiting
}

// Take blocks until a job is available, then returns it and clears the slot.
func (m *Mailbox[T]) Take() T {
	m.mu.Lock()
	defer m.mu.Unlock()

	for m.job == nil {
		m.cond.Wait()
	}

	j := *m.job
	m.job = nil
	return j
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
	return j
}

// HasJob reports whether a job is currently waiting.
func (m *Mailbox[T]) HasJob() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.job != nil
}
