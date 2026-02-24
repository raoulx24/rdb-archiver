package worker

import "context"

// queue.go provides a simple in-memory job queue used by the worker.
type Queue struct {
	ch chan Job
}

func NewQueue(size int) *Queue {
	return &Queue{ch: make(chan Job, size)}
}

func (q *Queue) Push(j Job) {
	q.ch <- j
}

func (q *Queue) Pop(ctx context.Context) (Job, bool) {
	select {
	case j := <-q.ch:
		return j, true
	case <-ctx.Done():
		return Job{}, false
	}
}
