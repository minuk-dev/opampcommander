// Package admin provides the implementation of the AdminUsecase interface.
package admin

import (
	"context"
	"fmt"
	"log/slog"

	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ applicationport.AdminUsecase = (*Service)(nil)

// Service is a struct that implements the AdminUsecase interface.
type Service struct {
	logger                   *slog.Logger
	agentUsecase             domainport.AgentUsecase
	connectionUsecase        domainport.ConnectionUsecase
	agentNotificationUsecase domainport.AgentNotificationUsecase
}

// New creates a new instance of the Service struct.
func New(
	agentUsecase domainport.AgentUsecase,
	connectionUsecase domainport.ConnectionUsecase,
	agentNotificationUsecase domainport.AgentNotificationUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		logger:                   logger,
		agentUsecase:             agentUsecase,
		connectionUsecase:        connectionUsecase,
		agentNotificationUsecase: agentNotificationUsecase,
	}
}

// ListConnections lists all connections.
func (s *Service) ListConnections(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Connection], error) {
	response, err := s.connectionUsecase.ListConnections(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}

	return response, nil
}
