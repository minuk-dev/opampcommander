// Package admin provides the implementation of the AdminUsecase interface.
package admin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"
	"k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ applicationport.AdminUsecase = (*Service)(nil)

// Service is a struct that implements the AdminUsecase interface.
type Service struct {
	logger                   *slog.Logger
	clock                    clock.Clock
	agentUsecase             agentport.AgentUsecase
	connectionUsecase        agentport.ConnectionUsecase
	agentNotificationUsecase agentport.AgentNotificationUsecase
}

// New creates a new instance of the Service struct.
func New(
	agentUsecase agentport.AgentUsecase,
	connectionUsecase agentport.ConnectionUsecase,
	agentNotificationUsecase agentport.AgentNotificationUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		logger:                   logger,
		clock:                    clock.RealClock{},
		agentUsecase:             agentUsecase,
		connectionUsecase:        connectionUsecase,
		agentNotificationUsecase: agentNotificationUsecase,
	}
}

// ListConnections lists connections filtered by namespace. The result is scoped to the
// server instance handling the request (connections are node-local live WebSockets); in
// HA use the agents API for a cluster-wide view of connectivity.
func (s *Service) ListConnections(
	ctx context.Context,
	namespace string,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Connection], error) {
	response, err := s.connectionUsecase.ListConnections(ctx, namespace, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}

	now := s.clock.Now()

	return v1.NewConnectionListResponse(
		lo.Map(response.Items, func(connection *agentmodel.Connection, _ int) v1.Connection {
			return v1.Connection{
				ID:                 connection.UID,
				InstanceUID:        connection.InstanceUID,
				Namespace:          connection.Namespace,
				Type:               connection.Type.String(),
				Alive:              connection.IsAlive(now),
				LastCommunicatedAt: v1.NewTime(connection.LastCommunicatedAt),
			}
		}),
		v1.ListMeta{
			RemainingItemCount: response.RemainingItemCount,
			Continue:           response.Continue,
		},
	), nil
}
