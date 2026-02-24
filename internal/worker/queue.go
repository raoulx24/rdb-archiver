package worker

import "context"

// provides a simple in-memory job queue used by the worker.

type Queue struct {
	Ch chan Job
}

func NewQueue(size int) *Queue {
	return &Queue{Ch: make(chan Job, size)}
}

func (q *Queue) Push(j Job) {
	q.Ch <- j
}

func (q *Queue) Pop(ctx context.Context) (Job, bool) {
	select {
	case j := <-q.Ch:
		return j, true
	case <-ctx.Done():
		return Job{}, false
	}
}
