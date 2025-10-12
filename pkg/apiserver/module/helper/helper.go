// Package helper provides utility functions and types to assist in building the application.
package helper

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// Controller is an interface that defines the methods for handling HTTP requests.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}

// Runner is an interface that defines a runner that can be started with a context.
type Runner interface {
	// Name returns the name of the runner.
	// It's only for debugging & logging purposes.
	Name() string

	// Run starts the runner with the given context.
	Run(ctx context.Context) error
}

// HealthIndicator is an interface that defines the methods for checking the health and readiness of the service.
type HealthIndicator interface {
	IsReady(ctx context.Context) bool
	IsHealth(ctx context.Context) bool
}

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
		fx.As(new(HealthIndicator)),
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
