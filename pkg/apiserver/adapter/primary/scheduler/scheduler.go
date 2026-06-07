// Package scheduler defines the Scheduler interface for long-running background
// runners that are started and supervised by the primary adapter's executor.
package scheduler

import (
	"context"
)

// Scheduler is an interface that defines a scheduler.
type Scheduler interface {
	// Name returns the name of the runner.
	// It's only for debugging & logging purposes.
	Name() string

	// Run starts the runner with the given context.
	// Run should synchronized & blocking method.
	Run(ctx context.Context) error
}
