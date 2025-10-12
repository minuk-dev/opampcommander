package observability

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	otelpromethues "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/minuk-dev/opampcommander/internal/management"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

func newMeterProvider(
	settings config.MetricSettings,
	logger *slog.Logger,
) (*metric.MeterProvider, management.RoutesInfo, error) {
	var (
		meterProvider *metric.MeterProvider
		routesInfo    management.RoutesInfo
		err           error
	)

	switch settings.Type {
	case config.MetricTypePrometheus:
		// Initialize Prometheus metrics
		var managementRoutesInfo management.RoutesInfo

		meterProvider, managementRoutesInfo, err = newPrometheusMetricProvider(
			settings.MetricSettingsForPrometheus.Path,
			logger,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize Prometheus metrics: %w", err)
		}

		routesInfo = append(routesInfo, managementRoutesInfo...)
	case config.MetricTypeOTel:
		return nil, nil, fmt.Errorf("%w: %s", ErrNoImplementation, settings.Type)
	default:
		return nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedObservabilityType, settings.Type)
	}

	// collect runtime metrics
	err = runtime.Start(
		runtime.WithMeterProvider(meterProvider),
	)
	if err != nil {
		// If runtime metrics cannot be started, we log the error but do not return it.
		logger.Warn("failed to start golang runtime metrics", slog.String("error", err.Error()))
	}

	return meterProvider, routesInfo, nil
}

func newPrometheusMetricProvider(
	path string,
	_ *slog.Logger,
) (*metric.MeterProvider, management.RoutesInfo, error) {
	registry := prometheus.NewRegistry()
	handler := createPrometheusHandler(registry)

	exporter, err := otelpromethues.New(
		otelpromethues.WithRegisterer(registry),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(exporter),
	)

	return provider, management.RoutesInfo{
		management.RouteInfo{
			Method:  http.MethodGet,
			Path:    path,
			Handler: handler,
		},
	}, nil
}

func createPrometheusHandler(registry *prometheus.Registry) http.Handler {
	var handlerOpts promhttp.HandlerOpts

	return promhttp.InstrumentMetricHandler(
		registry,
		promhttp.HandlerFor(registry, handlerOpts),
	)
}
