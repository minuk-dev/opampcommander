package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.AgentNotificationUsecase = (*AgentNotificationService)(nil)

// AgentNotificationService handles notifications about agent updates.
type AgentNotificationService struct {
	serverMessageUsecase   port.ServerMessageUsecase
	serverIdentityProvider port.ServerIdentityProvider
	logger                 *slog.Logger
}

// NewAgentNotificationService creates a new instance of AgentNotificationService.
func NewAgentNotificationService(
	serverMessageUsecase port.ServerMessageUsecase,
	serverIdentityProvider port.ServerIdentityProvider,
	logger *slog.Logger,
) *AgentNotificationService {
	return &AgentNotificationService{
		serverMessageUsecase:   serverMessageUsecase,
		serverIdentityProvider: serverIdentityProvider,
		logger:                 logger,
	}
}

// NotifyAgentUpdated notifies the connected server that the agent has pending messages.
func (s *AgentNotificationService) NotifyAgentUpdated(ctx context.Context, agent *model.Agent) error {
	if !agent.HasPendingServerMessages() || !agent.IsConnected(ctx) {
		return nil
	}

	server, err := agent.ConnectedServerID()
	if err != nil {
		s.logger.Warn("failed to notify agent update: cannot get connected server",
			slog.String("agentInstanceUID", agent.Metadata.InstanceUID.String()),
			slog.String("error", err.Error()),
		)

		return nil
	}

	currentServer, err := s.serverIdentityProvider.CurrentServer(ctx)
	if err != nil {
		s.logger.Warn("failed to notify agent update: cannot get current server",
			slog.String("agentInstanceUID", agent.Metadata.InstanceUID.String()),
			slog.String("error", err.Error()))

		currentServer = &model.Server{
			ID:              "unknown",
			LastHeartbeatAt: time.Time{},
			Conditions:      []model.ServerCondition{},
		}
	}

	err = s.serverMessageUsecase.SendMessageToServer(ctx, server, serverevent.Message{
		Source: currentServer.ID,
		Target: server.ID,
		Type:   serverevent.MessageTypeSendServerToAgent,
		Payload: serverevent.MessagePayload{
			MessageForServerToAgent: &serverevent.MessageForServerToAgent{
				TargetAgentInstanceUIDs: []uuid.UUID{
					agent.Metadata.InstanceUID,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send server messages to server %s for agent %s: %w",
			server.ID,
			agent.Metadata.InstanceUID,
			err,
		)
	}

	return nil
}

// RestartAgent requests the agent to restart.
func (s *AgentNotificationService) RestartAgent(ctx context.Context, instanceUID uuid.UUID) error {
	// For now, this is a placeholder implementation
	// In a real OpAMP implementation, this would send a restart command
	// to the agent through the OpAMP protocol
	s.logger.Info("restart requested for agent", "instanceUID", instanceUID.String())
	
	// TODO: Implement actual restart mechanism through OpAMP protocol
	// This might involve:
	// 1. Setting a restart flag in the agent's pending messages
	// 2. Sending ServerToAgent message with restart command
	// 3. Waiting for agent acknowledgment
	
	return nil
}
