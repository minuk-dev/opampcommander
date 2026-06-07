// Package helper provides FX annotation helpers and shared wiring utilities for
// the API server modules.
package helper

import (
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/scheduler"
	"github.com/minuk-dev/opampcommander/internal/management/healthcheck"
)

// AsHealthIndicator is a helper function to annotate a function as a health indicator.
func AsHealthIndicator(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(healthcheck.HealthIndicator)),
		fx.ResultTags(`group:"health_indicators"`),
	)
}

// AsRunner is a helper function to annotate a function as a scheduler runner.
// The annotated value is collected into the "runners" group and executed by the
// primary adapter's Executor.
func AsRunner(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(scheduler.Scheduler)),
		fx.ResultTags(`group:"runners"`),
	)
}

// PointerFunc is a generic function that returns a function that returns a pointer to the input value.
// It is a helper function to generate a function that returns a pointer to the input value.
// It is used to provide a function as a interface.
func PointerFunc[T any](a T) func() *T {
	return func() *T {
		return &a
	}
}

// ValueFunc is a generic function that returns a function that returns the input value.
// It is a helper function to generate a function that returns the input value.
func ValueFunc[T any](a T) func() T {
	return func() T {
		return a
	}
}
