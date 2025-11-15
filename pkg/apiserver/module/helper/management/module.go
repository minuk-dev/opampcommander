package management

import (
	"log/slog"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/minuk-dev/opampcommander/internal/management/healthcheck"
	"github.com/minuk-dev/opampcommander/internal/management/observability"
	"github.com/minuk-dev/opampcommander/internal/management/pprof"
)

// NewModule creates a new module for management services.
func NewModule() fx.Option {
	return fx.Module(
		"management",
		fx.Provide(
			// Management HTTP handlers
			observability.New, AsManagementHTTPHandler(Identity[*observability.Service]),
			pprof.NewHandler, AsManagementHTTPHandler(Identity[*pprof.Handler]),
			ExposeObservabilityComponents,

			// HTTP Client with tracing - must be provided after observability
			NewTracedHTTPClient,

			// Health checks
			fx.Annotate(healthcheck.NewHealthHelper, fx.ParamTags(`group:"health_indicators"`)),
			AsManagementHTTPHandler(Identity[*healthcheck.HealthHelper]),
			fx.Annotate(NewHTTPServer, fx.ParamTags(``, `group:"management_http_handlers"`)),
		),
		// Management HTTP server
		fx.Invoke(func(*HTTPServer) {}),
		// Logger
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger}
		}),
	)
}
