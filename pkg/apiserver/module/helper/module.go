package helper

import (
	"log/slog"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	internalhelper "github.com/minuk-dev/opampcommander/internal/helper"
	"github.com/minuk-dev/opampcommander/internal/management/healthcheck"
	"github.com/minuk-dev/opampcommander/internal/management/observability"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper/lifecycle"
	helpermanagement "github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper/management"
)

// NewModule creates a new module for helper services.
func NewModule() fx.Option {
	return fx.Module(
		"helper",
		fx.Provide(
			// Lifecycle management
			fx.Annotate(lifecycle.NewExecutor,
				fx.ParamTags(``, `group:"runners"`)),

			// Observability - ShutdownListener must be provided before observability.New
			internalhelper.NewShutdownListener,
			observability.New,
			helpermanagement.ExposeObservabilityComponents,
			fx.Annotate(
				helpermanagement.AsManagementHTTPHandler,
				fx.ResultTags(`group:"management_http_handlers"`),
			),

			// HTTP Client with tracing - must be provided after observability
			helpermanagement.NewTracedHTTPClient,

			// Security
			security.New,

			// Health checks
			fx.Annotate(healthcheck.NewHealthHelper,
				fx.ParamTags(`group:"health_indicators"`)),
			fx.Annotate(
				helpermanagement.AsHealthManagementHTTPHandler,
				fx.ResultTags(`group:"management_http_handlers"`),
			),

			// Management HTTP server
			fx.Annotate(helpermanagement.NewHTTPServer,
				fx.ParamTags(``, `group:"management_http_handlers"`)),
		),
		// Lifecycle hooks
		fx.Invoke(func(*lifecycle.Executor) {}),
		fx.Invoke(func(*helpermanagement.HTTPServer) {}),
		fx.Invoke(lifecycle.RegisterShutdownListener),

		// Logger
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger}
		}),
	)
}
