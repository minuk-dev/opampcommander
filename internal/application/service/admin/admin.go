// Package admin provides the implementation of the AdminUsecase interface.
package admin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ applicationport.AdminUsecase = (*Service)(nil)

// Service is a struct that implements the AdminUsecase interface.
type Service struct {
	logger            *slog.Logger
	agentUsecase      domainport.AgentUsecase
	commandUsecase    domainport.CommandUsecase
	connectionUsecase domainport.ConnectionUsecase
}

// New creates a new instance of the Service struct.
func New(
	agentUsecase domainport.AgentUsecase,
	commandUsecase domainport.CommandUsecase,
	connectionUsecase domainport.ConnectionUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		logger:            logger,
		agentUsecase:      agentUsecase,
		commandUsecase:    commandUsecase,
		connectionUsecase: connectionUsecase,
	}
}

// ApplyRawConfig applies the raw configuration to the target instance.
func (s *Service) ApplyRawConfig(ctx context.Context, targetInstanceUID uuid.UUID, config any) error {
	command := model.NewUpdateAgentConfigCommand(targetInstanceUID, config)

	err := s.commandUsecase.SaveCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("failed to save command: %w", err)
	}

	err = s.agentUsecase.UpdateAgentConfig(ctx, targetInstanceUID, config)
	if err != nil {
		return fmt.Errorf("failed to update agent config: %w", err)
	}

	return nil
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
