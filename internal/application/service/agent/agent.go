// Package agent provides application services for the agent
package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ applicationport.AgentManageUsecase = (*Service)(nil)

// Service is a struct that implements the AgentManageUsecase interface.
type Service struct {
	agentUsecase   domainport.AgentUsecase
	commandUsecase domainport.CommandUsecase
}

// New creates a new instance of the Service struct.
func New(
	agentUsecase domainport.AgentUsecase,
	commandUsecase domainport.CommandUsecase,
) *Service {
	return &Service{
		agentUsecase:   agentUsecase,
		commandUsecase: commandUsecase,
	}
}

// GetAgent implements port.AgentManageUsecase.
func (s *Service) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	agent, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return agent, nil
}

// ListAgents implements port.AgentManageUsecase.
func (s *Service) ListAgents(ctx context.Context) ([]*model.Agent, error) {
	agents, err := s.agentUsecase.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return agents, nil
}

// SendCommand implements port.AgentManageUsecase.
func (s *Service) SendCommand(ctx context.Context, targetInstanceUID uuid.UUID, command *model.Command) error {
	// apply command to agent
	err := s.agentUsecase.UpdateAgentConfig(ctx, targetInstanceUID, command.Data)
	if err != nil {
		return fmt.Errorf("failed to update agent config: %w", err)
	}

	// save command for audit log
	err = s.commandUsecase.SaveCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("failed to save command: %w", err)
	}

	return nil
}
