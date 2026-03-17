package snapshotwatcher

type Metrics interface {
	EventReceived()   // fsnotify/polling event arrived
	SnapshotParsed()  // snapshot successfully parsed
	InvalidSnapshot() // snapshot missing or unreadable
	JobEnqueued()     // job sent to mailbox
}

func (sw *Watcher) RebuildMetrics(newMetrics Metrics) {
	sw.logg.Debug("rebuilding metrics", "function", "RebuildMetrics")
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.metrics = newMetrics
}
