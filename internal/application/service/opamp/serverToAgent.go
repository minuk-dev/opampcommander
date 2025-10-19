package opamp

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
)

// fetchServerToAgent creates a ServerToAgent message from the agent.
func (s *Service) fetchServerToAgent(ctx context.Context, agentModel *model.Agent) *protobufs.ServerToAgent {
	var flags uint64

	// Request ReportFullState if:
	// 1. Agent has a pending ReportFullState command
	// 2. Agent's Metadata is not complete (missing description or capabilities)
	if agentModel.Commands.HasReportFullStateCommand() || !agentModel.Metadata.IsComplete() {
		flags |= uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState)
	}

	instanceUID := agentModel.Metadata.InstanceUID

	// Clear all commands after processing
	agentModel.Commands.Clear()

	// Build RemoteConfig if applicable
	remoteConfig := s.buildRemoteConfig(ctx, agentModel)

	//exhaustruct:ignore
	return &protobufs.ServerToAgent{
		InstanceUid:  instanceUID[:],
		Flags:        flags,
		RemoteConfig: remoteConfig,
	}
}

// buildRemoteConfig builds the remote configuration for the agent based on its agent groups.
func (s *Service) buildRemoteConfig(ctx context.Context, agentModel *model.Agent) *protobufs.AgentRemoteConfig {
	logger := s.logger.With(
		slog.String("method", "buildRemoteConfig"),
		slog.String("instanceUID", agentModel.Metadata.InstanceUID.String()),
	)

	// Check if agent supports RemoteConfig
	if !agentModel.Metadata.Capabilities.Has(agent.AgentCapabilityAcceptsRemoteConfig) {
		return nil
	}

	// Get agent groups for this agent
	agentGroups, err := s.agentGroupUsecase.GetAgentGroupsForAgent(ctx, agentModel)
	if err != nil {
		logger.Error("failed to get agent groups for agent", slog.String("error", err.Error()))
		return nil
	}

	// Find the first agent group with a non-nil AgentConfig
	// In case of multiple groups, we prioritize the first one
	for _, group := range agentGroups {
		if group.AgentConfig != nil && group.AgentConfig.Value != "" {
			logger.Info("applying remote config from agent group",
				slog.String("agentGroupName", group.Name),
			)

			// Build the AgentRemoteConfig message
			configMap := &protobufs.AgentConfigMap{
				ConfigMap: map[string]*protobufs.AgentConfigFile{
					"config": {
						Body: []byte(group.AgentConfig.Value),
					},
				},
			}

			return &protobufs.AgentRemoteConfig{
				Config: configMap,
			}
		}
	}

	return nil
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

// hasCapability checks if the agent has a specific capability.
func hasCapability(capabilities agent.Capabilities, capability agent.Capability) bool {
	return capabilities&agent.Capabilities(capability) != 0
}
