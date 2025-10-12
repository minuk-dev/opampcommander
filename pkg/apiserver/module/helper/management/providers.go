package management

import (
	"log/slog"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/management"
	"github.com/minuk-dev/opampcommander/internal/management/healthcheck"
	"github.com/minuk-dev/opampcommander/internal/management/observability"
)

// ObservabilityComponentResult exposes observability components to the DI container.
type ObservabilityComponentResult struct {
	fx.Out

	MeterProvider     metric.MeterProvider
	Logger            *slog.Logger
	TraceProvider     trace.TracerProvider
	TextMapPropagator propagation.TextMapPropagator
}

// ExposeObservabilityComponents extracts and exposes observability components from the service.
func ExposeObservabilityComponents(
	service *observability.Service,
) ObservabilityComponentResult {
	//exhaustruct:ignore
	return ObservabilityComponentResult{
		MeterProvider:     service.MeterProvider,
		Logger:            service.Logger,
		TraceProvider:     service.TraceProvider,
		TextMapPropagator: service.TextMapPropagator,
	}
}

// AsManagementHTTPHandler converts observability.Service to ManagementHTTPHandler.
//
//nolint:ireturn // This is a factory function that returns an interface by design.
func AsManagementHTTPHandler(svc *observability.Service) management.HTTPHandler {
	return svc
}

// AsHealthManagementHTTPHandler converts healthcheck.HealthHelper to ManagementHTTPHandler.
//
//nolint:ireturn // This is a factory function that returns an interface by design.
func AsHealthManagementHTTPHandler(helper *healthcheck.HealthHelper) management.HTTPHandler {
	return helper
}
