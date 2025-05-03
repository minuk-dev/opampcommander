package opamp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// FetchServerToAgent fetch a message.
func (s *Service) FetchServerToAgent(
	ctx context.Context,
	instanceUID uuid.UUID,
) (*protobufs.ServerToAgent, error) {
	s.logger.Info("FetchServerToAgent",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "start"),
	)

	agent, err := s.agentUsecase.GetOrCreateAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	serverToAgent, err := s.createServerToAgent(agent)
	if err != nil {
		s.logger.Error("FetchServerToAgent",
			slog.String("instanceUID", instanceUID.String()),
			slog.String("message", "failed to create server to agent"),
			slog.String("error", err.Error()),
		)

		return s.createFallbackServerToAgent(instanceUID), nil
	}

	err = s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		// Even if we failed to save the agent, we can still return the serverToAgent message.
		s.logger.Warn("FetchServerToAgent",
			slog.String("instanceUID", instanceUID.String()),
			slog.String("message", "failed to save agent"),
			slog.String("error", err.Error()),
		)
	}

	s.logger.Info("FetchServerToAgent",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "success"),
	)
	s.logger.Debug("FetchServerToAgent",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "serverToAgent"),
		slog.Any("serverToAgent", serverToAgent),
	)

	return serverToAgent, nil
}

// createServerToAgent creates a ServerToAgent message from the agent.
func (s *Service) createServerToAgent(agent *model.Agent) (*protobufs.ServerToAgent, error) {
	var flags uint64

	if agent == nil || agent.ReportFullState {
		flags |= uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState)
	}

	instanceUID := agent.InstanceUID

	err := agent.ResetByServerToAgent()
	if err != nil {
		return nil, fmt.Errorf("failed to reset agent: %w", err)
	}

	//exhaustruct:ignore
	return &protobufs.ServerToAgent{
		InstanceUid: instanceUID[:],
		Flags:       flags,
	}, nil
}

// createFallbackServerToAgent creates a fallback ServerToAgent message.
// This is used when the agent is not found or when there is an error in creating
// the ServerToAgent message.
func (s *Service) createFallbackServerToAgent(
	instanceUID uuid.UUID,
) *protobufs.ServerToAgent {
	//exhaustruct:ignore
	return &protobufs.ServerToAgent{
		InstanceUid: instanceUID[:],
	}
}
