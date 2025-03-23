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

type AgentService struct {
	agentPersistencePort port.AgentPersistencePort
}

func NewAgentService(
	agentPersistencePort port.AgentPersistencePort,
) *AgentService {
	return &AgentService{
		agentPersistencePort: agentPersistencePort,
	}
}

func (s *AgentService) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	agent, err := s.agentPersistencePort.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent from persistence: %w", err)
	}

	return agent, nil
}

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

func (s *AgentService) SaveAgent(ctx context.Context, agent *model.Agent) error {
	err := s.agentPersistencePort.PutAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to save agent to persistence: %w", err)
	}

	return nil
}

func (s *AgentService) ListAgents(ctx context.Context) ([]*model.Agent, error) {
	agents, err := s.agentPersistencePort.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return agents, nil
}
