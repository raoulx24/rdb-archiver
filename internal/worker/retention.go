package worker

import (
	"context"
	"log"
)

// defines a hook for invoking the retention engine after a snapshot.
// The real retention logic will be implemented later.

type Retention interface {
	Apply(ctx context.Context, archiveDir string) error
}

func (w *Worker) WithRetention(r Retention) *Worker {
	// In a real implementation, you'd store r and call it after Run().
	log.Printf("retention: attached retention engine (not yet wired)")
	return w
}
