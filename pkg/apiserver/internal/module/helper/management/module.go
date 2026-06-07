package management

import (
	"log/slog"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/observability"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/pprof"
)

// registerObservabilityShutdown hooks the observability Service's Shutdown into the
// FX lifecycle so trace/metric pipelines are flushed and released on stop. The
// observability package is FX-free, so the wiring lives here in the composition root.
func registerObservabilityShutdown(lifecycle fx.Lifecycle, service *observability.Service) {
	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop:  service.Shutdown,
	})
}

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
		// Flush observability pipelines on shutdown
		fx.Invoke(registerObservabilityShutdown),
		// Logger
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger}
		}),
	)
}
