// Package primary provides inbound (driving) adapters for the API server:
// the HTTP server with its controllers and the scheduler executor that runs
// background runners.
package primary

import (
	"go.uber.org/fx"
)

// New creates the primary adapter module.
func New() fx.Option {
	return fx.Options(
		NewHTTP(),
		fx.Provide(
			fx.Annotate(NewExecutor, fx.ParamTags(``, `group:"runners"`)),
		),
		fx.Invoke(func(*Executor) {}),
	)
}
