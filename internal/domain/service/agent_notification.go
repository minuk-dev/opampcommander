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
	serverUsecase          port.ServerUsecase
	serverIdentityProvider port.ServerIdentityProvider
	logger                 *slog.Logger
}

// NewAgentNotificationService creates a new instance of AgentNotificationService.
func NewAgentNotificationService(
	serverMessageUsecase port.ServerMessageUsecase,
	serverUsecase port.ServerUsecase,
	serverIdentityProvider port.ServerIdentityProvider,
	logger *slog.Logger,
) *AgentNotificationService {
	return &AgentNotificationService{
		serverMessageUsecase:   serverMessageUsecase,
		serverUsecase:          serverUsecase,
		serverIdentityProvider: serverIdentityProvider,
		logger:                 logger,
	}
}

// NotifyAgentUpdated notifies the connected server that the agent has pending messages.
func (s *AgentNotificationService) NotifyAgentUpdated(ctx context.Context, agent *model.Agent) error {
	logger := s.logger.With(
		slog.String("agentInstanceUID", agent.Metadata.InstanceUID.String()),
	)

	if !agent.HasPendingServerMessages() || !agent.IsConnected(ctx) {
		logger.Info("no notification sent: no pending messages or agent not connected")

		return nil
	}

	serverID, err := agent.ConnectedServerID()
	if err != nil {
		logger.Warn("failed to get connected server ID",
			slog.String("error", err.Error()),
		)

		return nil
	}

	currentServer, err := s.serverIdentityProvider.CurrentServer(ctx)
	if err != nil {
		logger.Warn("use unknown server because failed to get current server", slog.String("error", err.Error()))

		currentServer = s.getUnknownServer()
	}

	server, err := s.serverUsecase.GetServer(ctx, serverID)
	if err != nil {
		logger.Warn("failed to notify agent update: cannot get connected server",
			slog.String("serverID", serverID),
			slog.String("error", err.Error()),
		)

		return nil
	}

	err = s.serverMessageUsecase.SendMessageToServer(ctx, server, serverevent.Message{
		Source: currentServer.ID,
		Target: serverID,
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
			serverID,
			agent.Metadata.InstanceUID,
			err,
		)
	}

	return nil
}

// RestartAgent requests the agent to restart.
func (s *AgentNotificationService) RestartAgent(_ context.Context, instanceUID uuid.UUID) error {
	s.logger.Info("restart notification triggered", "instanceUID", instanceUID.String())
	// This method is now primarily for logging/monitoring purposes
	// The actual restart logic is handled in the application service layer
	return nil
}

func (s *AgentNotificationService) getUnknownServer() *model.Server {
	return &model.Server{
		ID:              "unknown",
		LastHeartbeatAt: time.Time{},
		Conditions:      []model.Condition{},
	}
}
