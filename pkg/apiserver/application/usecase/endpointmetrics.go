package usecase

import (
	"context"
	"time"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

// EndpointMetricsUsecase is a use case that reports how much telemetry collectors
// are sending to endpoints. The window is the rate window; a non-positive value
// asks the service to apply its configured default.
type EndpointMetricsUsecase interface {
	// GetEndpointThroughput reports the send throughput for a single endpoint.
	GetEndpointThroughput(ctx context.Context, namespace string, name string,
		window time.Duration) (*v1.EndpointThroughput, error)
	// ListEndpointThroughput reports the send throughput for every endpoint in a
	// namespace.
	ListEndpointThroughput(ctx context.Context, namespace string,
		window time.Duration) (*v1.ListResponse[v1.EndpointThroughput], error)
}
