// Package observability provides observability features for the application.
package observability

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	metricapi "go.opentelemetry.io/otel/metric"
	traceapi "go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

var (
	// ErrUnsupportedObservabilityType is returned when an unsupported observability type is specified.
	ErrUnsupportedObservabilityType = errors.New("unsupported observability type")

	// ErrNoImplementation is returned when no implementation is provided for the observability type.
	ErrNoImplementation = errors.New("no implementation provided for the observability type")

	// ErrInvalidPrometheusEndpoint is returned when the Prometheus endpoint is invalid.
	ErrInvalidPrometheusEndpoint = errors.New("invalid Prometheus endpoint URL")
)

// Service provides observability features such as metrics and tracing.
type Service struct {
	serviceName   string // observability service name
	meterProvider metricapi.MeterProvider
	traceProvider traceapi.TracerProvider
	logger        *slog.Logger
}

// New creates a new observability Service based on the provided settings.
// It provides observability service and logger as dependencies.
func New(
	settings *config.ObservabilitySettings,
	lifecycle fx.Lifecycle,
) (*Service, *slog.Logger, error) {
	logger, err := newLogger(settings)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create logger: %w", err)
	}

	if settings == nil {
		// If no settings are provided, return a default Service instance.
		//exhaustruct:ignore
		return &Service{}, logger, nil
	}

	service := &Service{
		serviceName:   settings.ServiceName,
		meterProvider: nil,
		traceProvider: nil,
		logger:        logger,
	}

	if settings.Metric.Enabled {
		switch settings.Metric.Type {
		case config.MetricTypePrometheus:
			// Initialize Prometheus metrics
			service.meterProvider, err = newPrometheusMetricProvider(settings.Metric.Endpoint, lifecycle, service.logger)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to initialize Prometheus metrics: %w", err)
			}
		case config.MetricTypeOTel:
			return nil, nil, fmt.Errorf("%w: %s", ErrNoImplementation, settings.Metric.Type)
		default:
			return nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedObservabilityType, settings.Metric.Type)
		}

		// collect runtime metrics
		err = runtime.Start(
			runtime.WithMeterProvider(service.meterProvider),
		)
		if err != nil {
			// If runtime metrics cannot be started, we log the error but do not return it.
			logger.Warn("failed to start golang runtime metrics", slog.String("error", err.Error()))
		}
	}

	if settings.Trace.Enabled {
		service.traceProvider, err = newTraceProvider(lifecycle, settings.Trace)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize trace provider: %w", err)
		}
	}

	return service, logger, nil
}

// Middleware returns a Gin middleware function that applies OpenTelemetry instrumentation.
func (service *Service) Middleware() gin.HandlerFunc {
	if service.meterProvider == nil {
		return func(ctx *gin.Context) {
			ctx.Next()
		}
	}

	var opts []otelgin.Option

	if service.meterProvider != nil {
		opts = append(opts, otelgin.WithMeterProvider(service.meterProvider))
	}

	if service.traceProvider != nil {
		opts = append(opts, otelgin.WithTracerProvider(service.traceProvider))
	}

	return otelgin.Middleware(service.serviceName, opts...)
}
