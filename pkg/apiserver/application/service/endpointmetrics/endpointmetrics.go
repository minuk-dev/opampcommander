// Package endpointmetrics provides the application service that reports how much
// telemetry collectors are sending to endpoints.
package endpointmetrics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.EndpointMetricsUsecase = (*Service)(nil)

// fallbackWindow is the rate window used when neither the caller nor the
// configuration provides one.
const fallbackWindow = 5 * time.Minute

// Service reports endpoint throughput, delegating aggregation to the domain
// usecase and mapping results to API models.
type Service struct {
	usecase       agentport.EndpointMetricsUsecase
	mapper        *helper.Mapper
	clock         clock.Clock
	defaultWindow time.Duration
	logger        *slog.Logger
}

// NewEndpointMetricsService creates a new endpoint-metrics Service. defaultWindow
// is the rate window applied when a request does not specify one; a non-positive
// value falls back to a built-in default.
func NewEndpointMetricsService(
	usecase agentport.EndpointMetricsUsecase,
	defaultWindow time.Duration,
	logger *slog.Logger,
) *Service {
	realClock := clock.NewRealClock()

	return &Service{
		usecase:       usecase,
		mapper:        helper.NewMapper(realClock, 0),
		clock:         realClock,
		defaultWindow: defaultWindow,
		logger:        logger,
	}
}

// GetEndpointThroughput implements [port.EndpointMetricsUsecase].
func (s *Service) GetEndpointThroughput(
	ctx context.Context,
	namespace string,
	name string,
	window time.Duration,
) (*v1.EndpointThroughput, error) {
	throughput, err := s.usecase.GetEndpointThroughput(
		ctx, namespace, name, s.resolveWindow(window), s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("get endpoint throughput: %w", err)
	}

	return s.mapper.MapEndpointThroughputToAPI(throughput), nil
}

// ListEndpointThroughput implements [port.EndpointMetricsUsecase].
func (s *Service) ListEndpointThroughput(
	ctx context.Context,
	namespace string,
	window time.Duration,
) (*v1.ListResponse[v1.EndpointThroughput], error) {
	throughputs, err := s.usecase.ListEndpointThroughput(
		ctx, namespace, s.resolveWindow(window), s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("list endpoint throughput: %w", err)
	}

	return &v1.ListResponse[v1.EndpointThroughput]{
		Kind:       v1.EndpointThroughputKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.ListMeta{Continue: "", RemainingItemCount: 0},
		Items: lo.Map(throughputs, func(item *agentmodel.EndpointThroughput, _ int) v1.EndpointThroughput {
			return *s.mapper.MapEndpointThroughputToAPI(item)
		}),
	}, nil
}

// resolveWindow applies the configured default (or the built-in fallback) when
// the caller does not supply a positive window.
func (s *Service) resolveWindow(window time.Duration) time.Duration {
	if window > 0 {
		return window
	}

	if s.defaultWindow > 0 {
		return s.defaultWindow
	}

	return fallbackWindow
}
