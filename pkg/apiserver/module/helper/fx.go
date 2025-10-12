package helper

import (
	"github.com/minuk-dev/opampcommander/internal/management/healthcheck"
	"go.uber.org/fx"
)

// AsController is a helper function to annotate a function as a controller.
func AsController(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Controller)),
		fx.ResultTags(`group:"controllers"`),
	)
}

// AsHealthIndicator is a helper function to annotate a function as a health indicator.
func AsHealthIndicator(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(healthcheck.HealthIndicator)),
		fx.ResultTags(`group:"health_indicators"`),
	)
}

// AsRunner is a helper function to annotate a function as a runner.
func AsRunner(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Runner)),
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

// Identity is a generic function that returns the input value.
// It is a helper function to generate a function that returns the input value.
// It is used to provide a function as a interface.
func Identity[T any](a T) T {
	return a
}
