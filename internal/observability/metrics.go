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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	otelpromethues "go.opentelemetry.io/otel/exporters/prometheus"
	metricapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

const (
	// DefaultPrometheusReadTimeout is the default timeout for reading Prometheus metrics.
	DefaultPrometheusReadTimeout = 30 * time.Second

	// DefaultPrometheusReadHeaderTimeout is the default timeout for reading Prometheus headers.
	DefaultPrometheusReadHeaderTimeout = 10 * time.Second
)

//nolint:ireturn
func newMeterProvider(
	lifecycle fx.Lifecycle,
	settings config.MetricSettings,
	logger *slog.Logger,
) (metricapi.MeterProvider, error) {
	var (
		meterProvider metricapi.MeterProvider
		err           error
	)

	switch settings.Type {
	case config.MetricTypePrometheus:
		// Initialize Prometheus metrics
		meterProvider, err = newPrometheusMetricProvider(settings.Endpoint, lifecycle, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Prometheus metrics: %w", err)
		}
	case config.MetricTypeOTel:
		return nil, fmt.Errorf("%w: %s", ErrNoImplementation, settings.Type)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedObservabilityType, settings.Type)
	}

	// collect runtime metrics
	err = runtime.Start(
		runtime.WithMeterProvider(meterProvider),
	)
	if err != nil {
		// If runtime metrics cannot be started, we log the error but do not return it.
		logger.Warn("failed to start golang runtime metrics", slog.String("error", err.Error()))
	}

	return meterProvider, nil
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

	setupMetricsLifecycleHooks(lifecycle, server, logger)

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

func setupMetricsLifecycleHooks(lifecycle fx.Lifecycle, server *http.Server, logger *slog.Logger) {
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
