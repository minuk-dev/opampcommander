package helper

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	internalhelper "github.com/minuk-dev/opampcommander/internal/helper"
	"github.com/minuk-dev/opampcommander/internal/observability"
	"github.com/minuk-dev/opampcommander/internal/security"
)

// NewModule creates a new module for helper services.
func NewModule() fx.Option {
	return fx.Module(
		"helper",
		fx.Provide(
			// executor for runners
			fx.Annotate(NewExecutor,
				fx.ParamTags(``, `group:"runners"`)),
			// security,
			security.New,
			// ShutdownListener must be provided before observability.New
			internalhelper.NewShutdownListener,
			observability.New,
			exposeObservabilityComponents,
			fx.Annotate(
				func(svc *observability.Service) observability.ManagementHTTPHandler {
					return svc
				},
				fx.ResultTags(`group:"management_http_handlers"`),
			),
			fx.Annotate(observability.NewHealthHelper,
				fx.ParamTags(`group:"health_indicators"`)),
			fx.Annotate(
				func(helper *observability.HealthHelper) observability.ManagementHTTPHandler {
					return helper
				},
				fx.ResultTags(`group:"management_http_handlers"`),
			),
			fx.Annotate(NewManagementHTTPServer,
				fx.ParamTags(``, `group:"management_http_handlers"`)),
		),
		// invoke
		fx.Invoke(func(*Executor) {}),
		fx.Invoke(func(*ManagementHTTPServer) {}),
		fx.Invoke(invokeShutdownListener),
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{
				Logger: logger,
			}
		}),
	)
}

func invokeShutdownListener(
	shutdownlistener *internalhelper.ShutdownListener,
	lifecycle fx.Lifecycle,
) {
	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			err := shutdownlistener.Shutdown(ctx)
			if err != nil {
				return fmt.Errorf("error during shutdown: %w", err)
			}
			return nil
		},
	})
}

type observabilityComponentResult struct {
	fx.Out

	MeterProvider     metric.MeterProvider
	Logger            *slog.Logger
	TraceProvider     trace.TracerProvider
	TextMapPropagator propagation.TextMapPropagator
}

func exposeObservabilityComponents(
	service *observability.Service,
) observabilityComponentResult {
	return observabilityComponentResult{
		MeterProvider:     service.MeterProvider,
		Logger:            service.Logger,
		TraceProvider:     service.TraceProvider,
		TextMapPropagator: service.TextMapPropagator,
	}
}
