package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/remoteconfig"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.AgentUsecase = (*AgentService)(nil)

// AgentService is a struct that implements the AgentUsecase interface.
type AgentService struct {
	agentPersistencePort port.AgentPersistencePort
}

// NewAgentService creates a new instance of AgentService.
func NewAgentService(
	agentPersistencePort port.AgentPersistencePort,
) *AgentService {
	return &AgentService{
		agentPersistencePort: agentPersistencePort,
	}
}

// GetAgent retrieves an agent by its instance UID.
func (s *AgentService) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	agent, err := s.agentPersistencePort.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent from persistence: %w", err)
	}

	return agent, nil
}

// GetOrCreateAgent retrieves an agent by its instance UID.
func (s *AgentService) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	agent, err := s.GetAgent(ctx, instanceUID)
	if err != nil {
		if errors.Is(err, port.ErrAgentNotExist) {
			agent = &model.Agent{
				InstanceUID:         instanceUID,
				Capabilities:        nil,
				Description:         nil,
				EffectiveConfig:     nil,
				PackageStatuses:     nil,
				ComponentHealth:     nil,
				RemoteConfig:        remoteconfig.New(),
				CustomCapabilities:  nil,
				AvailableComponents: nil,
			}
		} else {
			return nil, fmt.Errorf("failed to get agent: %w", err)
		}
	}

	return agent, nil
}

// SaveAgent saves the agent to the persistence layer.
func (s *AgentService) SaveAgent(ctx context.Context, agent *model.Agent) error {
	err := s.agentPersistencePort.PutAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to save agent to persistence: %w", err)
	}

	return nil
}

// ListAgents retrieves all agents from the persistence layer.
func (s *AgentService) ListAgents(ctx context.Context) ([]*model.Agent, error) {
	agents, err := s.agentPersistencePort.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return agents, nil
}

// UpdateAgentConfig updates the agent configuration.
func (s *AgentService) UpdateAgentConfig(ctx context.Context, instanceUID uuid.UUID, config any) error {
	agent, err := s.GetOrCreateAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get or create agent: %w", err)
	}

	err = agent.ApplyRemoteConfig(config)
	if err != nil {
		return fmt.Errorf("failed to apply remote config: %w", err)
	}

	err = s.SaveAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to save agent: %w", err)
	}

	return nil
}
