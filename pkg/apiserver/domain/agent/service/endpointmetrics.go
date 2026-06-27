package agentservice

import (
	"context"
	"fmt"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.EndpointMetricsUsecase = (*EndpointMetricsService)(nil)

// endpointThroughputPageSize is the number of endpoints fetched per page when
// aggregating a whole namespace.
const endpointThroughputPageSize = 100

// endpointThroughputMaxPages caps pagination so a misbehaving Continue token can
// never spin forever.
const endpointThroughputMaxPages = 1000

// EndpointMetricsService aggregates endpoint throughput by combining the endpoint
// store with the metrics-query port.
type EndpointMetricsService struct {
	persistence agentport.EndpointPersistencePort
	metrics     agentport.EndpointMetricsQueryPort
}

// NewEndpointMetricsService creates a new EndpointMetricsService.
func NewEndpointMetricsService(
	persistence agentport.EndpointPersistencePort,
	metrics agentport.EndpointMetricsQueryPort,
) *EndpointMetricsService {
	return &EndpointMetricsService{
		persistence: persistence,
		metrics:     metrics,
	}
}

// GetEndpointThroughput implements [agentport.EndpointMetricsUsecase].
func (s *EndpointMetricsService) GetEndpointThroughput(
	ctx context.Context,
	namespace string,
	name string,
	window time.Duration,
	evaluatedAt time.Time,
) (*agentmodel.EndpointThroughput, error) {
	endpoint, err := s.persistence.GetEndpoint(ctx, namespace, name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}

	throughput, err := s.metrics.QueryEndpointThroughput(ctx, endpoint, window, evaluatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to query endpoint throughput: %w", err)
	}

	return throughput, nil
}

// ListEndpointThroughput implements [agentport.EndpointMetricsUsecase]. It pages
// through every endpoint in the namespace and queries each one's throughput.
func (s *EndpointMetricsService) ListEndpointThroughput(
	ctx context.Context,
	namespace string,
	window time.Duration,
	evaluatedAt time.Time,
) ([]*agentmodel.EndpointThroughput, error) {
	result := make([]*agentmodel.EndpointThroughput, 0)
	continueToken := ""

	for range endpointThroughputMaxPages {
		//exhaustruct:ignore // only paging fields are relevant; defaults exclude deleted.
		resp, err := s.persistence.ListEndpoints(ctx, namespace, &model.ListOptions{
			Limit:    endpointThroughputPageSize,
			Continue: continueToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list endpoints: %w", err)
		}

		for _, endpoint := range resp.Items {
			throughput, err := s.metrics.QueryEndpointThroughput(ctx, endpoint, window, evaluatedAt)
			if err != nil {
				return nil, fmt.Errorf("failed to query throughput for endpoint %q: %w",
					endpoint.Metadata.Name, err)
			}

			result = append(result, throughput)
		}

		if resp.Continue == "" {
			break
		}

		continueToken = resp.Continue
	}

	return result, nil
}
