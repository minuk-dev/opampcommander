package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.AgentUsecase = (*AgentService)(nil)

// AgentService is a struct that implements the AgentUsecase interface.
type AgentService struct {
	agentPersistencePort   port.AgentPersistencePort
	serverMessageUsecase   port.ServerMessageUsecase
	serverIdentityProvider port.ServerIdentityProvider
	logger                 *slog.Logger
}

// NewAgentService creates a new instance of AgentService.
func NewAgentService(
	agentPersistencePort port.AgentPersistencePort,
	serverMessageUsecase port.ServerMessageUsecase,
	serverIdentityProvider port.ServerIdentityProvider,
	logger *slog.Logger,
) *AgentService {
	return &AgentService{
		agentPersistencePort:   agentPersistencePort,
		serverMessageUsecase:   serverMessageUsecase,
		serverIdentityProvider: serverIdentityProvider,
		logger:                 logger,
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
// If the agent doesn't exist, it creates a new one with default values.
func (s *AgentService) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	agent, err := s.GetAgent(ctx, instanceUID)
	if err != nil {
		if errors.Is(err, port.ErrResourceNotExist) {
			agent = model.NewAgent(instanceUID)
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

	if agent.HasPendingServerMessages() && agent.IsConnected(ctx) {
		server, err := agent.ConnectedServer()
		if err != nil {
			s.logger.Warn("saved agent but failed to send server messages: cannot get connected server",
				slog.String("agentInstanceUID", agent.Metadata.InstanceUID.String()),
				slog.String("error", err.Error()),
			)
			return nil
		}

		currentServer, err := s.serverIdentityProvider.CurrentServer(ctx)
		if err != nil {
			s.logger.Warn("saved agent but failed to send server messages: cannot get current server",
				slog.String("agentInstanceUID", agent.Metadata.InstanceUID.String()),
				slog.String("error", err.Error()))

			currentServer = &model.Server{
				ID: "unknown",
			}
		}

		s.serverMessageUsecase.SendMessageToServer(ctx, server, serverevent.Message{
			Source: currentServer.ID,
			Type:   serverevent.MessageTypeSendServerToAgent,
			Payload: serverevent.MessagePayload{
				MessageForServerToAgent: &serverevent.MessageForServerToAgent{
					TargetAgentInstanceUIDs: []uuid.UUID{
						agent.Metadata.InstanceUID,
					},
				},
			},
		})
	}

	return nil
}

// ListAgents retrieves all agents from the persistence layer.
func (s *AgentService) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	res, err := s.agentPersistencePort.ListAgents(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return res, nil
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

// ListAgentsBySelector implements port.AgentUsecase.
func (s *AgentService) ListAgentsBySelector(
	ctx context.Context,
	selector model.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	resp, err := s.agentPersistencePort.ListAgentsBySelector(ctx, selector, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents by selector: %w", err)
	}

	return resp, nil
}
