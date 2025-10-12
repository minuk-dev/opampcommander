// Package observability provides observability features for the application.
package observability

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	metricapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	traceapi "go.opentelemetry.io/otel/trace"

	"github.com/minuk-dev/opampcommander/internal/helper"
	"github.com/minuk-dev/opampcommander/internal/management"
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

var (
	_ management.HTTPHandler = (*Service)(nil)
)

// Service provides observability features such as metrics and tracing.
type Service struct {
	serviceName string // observability service name

	routeInfos management.RoutesInfo // some observability types may provide management routes

	MeterProvider     metricapi.MeterProvider
	TraceProvider     traceapi.TracerProvider
	TextMapPropagator propagation.TextMapPropagator
	Logger            *slog.Logger
}

// New creates a new observability Service based on the provided settings.
func New(
	settings *config.ObservabilitySettings,
	shutdownlistner *helper.ShutdownListener,
) (*Service, error) {
	logger, err := newLogger(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	if settings == nil {
		// If no settings are provided, return a default Service instance.
		//exhaustruct:ignore
		return &Service{
			Logger: logger,
		}, nil
	}

	//exhaustruct:ignore
	service := &Service{
		serviceName:       settings.ServiceName,
		MeterProvider:     nil,
		TraceProvider:     nil,
		TextMapPropagator: nil,
		Logger:            logger,
	}

	if settings.Metric.Enabled {
		meterProvider, managementRoutesInfo, err := newMeterProvider(
			settings.Metric,
			logger,
		)
		if err != nil {
			logger.Warn("failed to initialize meter provider",
				slog.String("error", err.Error()),
			)
			// If meter provider cannot be initialized, we log the error but do not return it.
		}

		service.MeterProvider = meterProvider
		service.routeInfos = append(service.routeInfos, managementRoutesInfo...)
	}

	if settings.Trace.Enabled {
		service.TraceProvider, err = newTraceProvider(
			service.serviceName,
			shutdownlistner,
			settings.Trace,
			logger,
		)
		if err != nil {
			logger.Warn("failed to initialize trace provider", slog.String("error", err.Error()))
			// If trace provider cannot be initialized, we log the error but do not return it.
		}

		service.TextMapPropagator = propagation.TraceContext{}
	}

	return service, nil
}

// RoutesInfos returns the management routes provided by this service.
func (service *Service) RoutesInfos() management.RoutesInfo {
	return service.routeInfos
}

// Middleware returns a Gin middleware function that applies OpenTelemetry instrumentation.
func (service *Service) Middleware() gin.HandlerFunc {
	if service.MeterProvider == nil {
		return func(ctx *gin.Context) {
			ctx.Next()
		}
	}

	var opts []otelgin.Option

	if service.MeterProvider != nil {
		opts = append(opts, otelgin.WithMeterProvider(service.MeterProvider))
	}

	if service.TraceProvider != nil {
		opts = append(opts, otelgin.WithTracerProvider(service.TraceProvider))
	}

	if service.TextMapPropagator != nil {
		opts = append(opts, otelgin.WithPropagators(service.TextMapPropagator))
	}

	return otelgin.Middleware(service.serviceName, opts...)
}
