// Package observability provides observability features for the application.
package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	otelpromethues "go.opentelemetry.io/otel/exporters/prometheus"
	metricapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
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

const (
	// DefaultPrometheusReadTimeout is the default timeout for reading Prometheus metrics.
	DefaultPrometheusReadTimeout = 30 * time.Second

	// DefaultPrometheusReadHeaderTimeout is the default timeout for reading Prometheus headers.
	DefaultPrometheusReadHeaderTimeout = 10 * time.Second
)

// Service provides observability features such as metrics and tracing.
type Service struct {
	serviceName   string // observability service name
	meterProvider metricapi.MeterProvider
	traceProvider traceapi.TracerProvider
}

// New creates a new observability Service based on the provided settings.
func New(
	settings *config.ObservabilitySettings,
	lifecycle fx.Lifecycle,
	logger *slog.Logger,
) (*Service, error) {
	if settings == nil {
		// If no settings are provided, return a default Service instance.
		//exhaustruct:ignore
		return &Service{}, nil
	}

	service := &Service{
		serviceName:   settings.ServiceName,
		meterProvider: nil,
		traceProvider: nil,
	}

	var err error

	if settings.Metric.Enabled {
		switch settings.Metric.Type {
		case config.MetricTypePrometheus:
			// Initialize Prometheus metrics
			service.meterProvider, err = newPrometheusMetricProvider(settings.Metric.Endpoint, lifecycle, logger)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize Prometheus metrics: %w", err)
			}
		case config.MetricTypeOTel:
			return nil, fmt.Errorf("%w: %s", ErrNoImplementation, settings.Metric.Type)
		default:
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedObservabilityType, settings.Metric.Type)
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
		return nil, ErrNoImplementation
	}

	return service, nil
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

func newPrometheusMetricProvider(
	endpoint string,
	lifecycle fx.Lifecycle,
	logger *slog.Logger,
) (*metric.MeterProvider, error) {
	url, err := validatePrometheusEndpoint(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to validate Prometheus endpoint: %w", err)
	}

	registry := prometheus.NewRegistry()
	handler := createPrometheusHandler(registry)

	server := createPrometheusServer(url, handler)

	setupLifecycleHooks(lifecycle, server, logger)

	exporter, err := otelpromethues.New(
		otelpromethues.WithRegisterer(registry),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(exporter),
	)

	return provider, nil
}

func validatePrometheusEndpoint(endpoint string) (*url.URL, error) {
	url, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid Prometheus endpoint URL: %w", err)
	}

	if url.Scheme != "http" && url.Scheme != "" {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPrometheusEndpoint, url.Scheme)
	}

	if url.Host == "" {
		return nil, fmt.Errorf("%w: missing host in Prometheus endpoint URL", ErrInvalidPrometheusEndpoint)
	}

	return url, nil
}

func createPrometheusHandler(registry *prometheus.Registry) http.Handler {
	var handlerOpts promhttp.HandlerOpts

	return promhttp.InstrumentMetricHandler(registry, promhttp.HandlerFor(registry, handlerOpts))
}

func createPrometheusServer(url *url.URL, handler http.Handler) *http.Server {
	//exhaustruct:ignore
	return &http.Server{
		Addr: url.Host,
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodGet {
				http.Error(writer, "Method Not Allowed", http.StatusMethodNotAllowed)

				return
			}
			if req.URL.Path != url.Path {
				http.NotFound(writer, req)

				return
			}
			handler.ServeHTTP(writer, req)
		}),
		ReadTimeout:       DefaultPrometheusReadTimeout,
		ReadHeaderTimeout: DefaultPrometheusReadHeaderTimeout,
	}
}

func setupLifecycleHooks(lifecycle fx.Lifecycle, server *http.Server, logger *slog.Logger) {
	var httpWg sync.WaitGroup

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			httpWg.Add(1)
			go func() {
				defer httpWg.Done()
				err := server.ListenAndServe()
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Warn("Failed to start Prometheus metrics server", slog.String("error", err.Error()))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			err := server.Shutdown(ctx)
			if err != nil {
				return fmt.Errorf("failed to shutdown Prometheus metrics server: %w", err)
			}
			httpWg.Wait()

			return nil
		},
	})
}
