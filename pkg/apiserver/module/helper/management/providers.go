package management

import (
	"log/slog"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
func AsManagementHTTPHandler(svc *observability.Service) management.HTTPHandler {
	return svc
}

// AsHealthManagementHTTPHandler converts healthcheck.HealthHelper to ManagementHTTPHandler.
func AsHealthManagementHTTPHandler(helper *healthcheck.HealthHelper) management.HTTPHandler {
	return helper
}

// NewTracedHTTPClient creates an HTTP client with OpenTelemetry tracing instrumentation.
// If TracerProvider is nil, it returns the standard http.DefaultClient.
func NewTracedHTTPClient(tracerProvider trace.TracerProvider) *http.Client {
	if tracerProvider == nil {
		return http.DefaultClient
	}

	//exhaustruct:ignore
	return &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithTracerProvider(tracerProvider),
		),
	}
}
