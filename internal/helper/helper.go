// Package helper provides utility functions and types for the application layer.
package helper

import "context"

// Runner is an interface that defines a runner that can be started with a context.
type Runner interface {
	// Run starts the runner with the given context.
	Run(ctx context.Context) error
}
