package worker

import (
	"context"
	"log"
)

// contains the loop that continuously pulls jobs from the queue
// and executes them using the Worker.

func RunLoop(ctx context.Context, w *Worker, q *Queue) {
	for {
		job, ok := q.Pop(ctx)
		if !ok {
			return
		}

		_, err := w.Run(ctx, job.SourcePath)
		if err != nil {
			log.Printf("worker: failed to process %s: %v", job.SourcePath, err)
		}
	}
}
