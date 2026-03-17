package mailbox

type Metrics interface {
	JobEnqueued()    // a job was put into the mailbox
	JobOverwritten() // a job was replaced because mailbox was full
	JobDequeued()    // worker took the job
}

func (m *Mailbox[T]) RebuildMetrics(newMetrics Metrics) {
	m.logg.Debug("rebuilding metrics", "function", "RebuildMetrics")
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = newMetrics
}
