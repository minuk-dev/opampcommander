// Package helper provides utility functions and types for the application layer.
package helper

import "context"

// Runner is an interface that defines a runner that can be started with a context.
type Runner interface {
	// Name returns the name of the runner.
	// It's only for debugging & logging purposes.
	Name() string

	// Run starts the runner with the given context.
	Run(ctx context.Context) error
}
