// Package metrics provides outbound adapters for querying a metrics backend for
// endpoint throughput, plus a no-op fallback used when no backend is configured.
package metrics

import (
	"context"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.EndpointMetricsQueryPort = (*NoopAdapter)(nil)

// NoopAdapter is an [agentport.EndpointMetricsQueryPort] that measures nothing.
// It is wired when no metrics backend is configured (e.g. standalone mode) so
// the port is always satisfiable; every signal comes back unmeasured.
type NoopAdapter struct{}

// NewNoopAdapter creates a NoopAdapter.
func NewNoopAdapter() *NoopAdapter {
	return &NoopAdapter{}
}

// QueryEndpointThroughput returns a result with every signal unmeasured.
func (a *NoopAdapter) QueryEndpointThroughput(
	_ context.Context,
	endpoint *agentmodel.Endpoint,
	window time.Duration,
	at time.Time,
) (*agentmodel.EndpointThroughput, error) {
	return &agentmodel.EndpointThroughput{
		Namespace:   endpoint.Metadata.Namespace,
		Name:        endpoint.Metadata.Name,
		EvaluatedAt: at,
		Window:      window,
		Metrics:     agentmodel.SignalThroughput{Measured: false, PerSecond: 0, SeriesCount: 0},
		Logs:        agentmodel.SignalThroughput{Measured: false, PerSecond: 0, SeriesCount: 0},
		Traces:      agentmodel.SignalThroughput{Measured: false, PerSecond: 0, SeriesCount: 0},
	}, nil
}
