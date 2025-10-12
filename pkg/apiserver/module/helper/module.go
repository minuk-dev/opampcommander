package helper

import (
	"log/slog"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/minuk-dev/opampcommander/internal/observability"
	"github.com/minuk-dev/opampcommander/internal/security"
)

// NewModule creates a new module for helper services.
func NewModule() fx.Option {
	return fx.Module(
		"helper",
		fx.Provide(
			// executor for runners
			fx.Annotate(NewExecutor, fx.ParamTags("", `group:"runners"`)),
			// security,
			security.New,
			observability.New,
		),
		// invoke
		fx.Invoke(func(*Executor) {}),
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{
				Logger: logger,
			}
		}),
	)
}
