package retention

import "context"

// Defines the interface for retention policies

type Engine interface {
	Apply(ctx context.Context, archiveDir string) error
}
