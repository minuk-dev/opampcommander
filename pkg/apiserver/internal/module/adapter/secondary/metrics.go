package secondary

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/metrics"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/metrics/prometheus"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

// errMetricsBackendAddressEmpty is returned when a metrics backend is selected
// but no address is configured for it.
var errMetricsBackendAddressEmpty = errors.New("metrics backend address is empty")

// newEndpointMetricsQueryAdapter selects the endpoint-throughput query adapter
// from the configured metrics backend: a Prometheus-compatible client when
// configured, otherwise a no-op so the port is always satisfiable.
func newEndpointMetricsQueryAdapter(
	settings *config.ServerSettings,
	logger *slog.Logger,
) (agentport.EndpointMetricsQueryPort, error) {
	backend := settings.MetricsBackend

	if backend.Type == config.MetricsBackendTypePrometheus {
		if backend.Address == "" {
			return nil, fmt.Errorf("%w (type %q)", errMetricsBackendAddressEmpty, backend.Type)
		}

		adapter, err := prometheus.NewAdapter(backend.Address, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create prometheus metrics adapter: %w", err)
		}

		return adapter, nil
	}

	return metrics.NewNoopAdapter(), nil
}
