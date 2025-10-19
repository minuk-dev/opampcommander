package opamp

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model/remoteconfig"
	"github.com/minuk-dev/opampcommander/internal/domain/model/vo"
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
			configBytes := []byte(group.AgentConfig.Value)

			// Compute hash of the configuration
			configHash, err := vo.NewHash(configBytes)
			if err != nil {
				logger.Error("failed to compute config hash", slog.String("error", err.Error()))

				return nil
			}

			// Check if agent already has this config applied
			currentStatus := agentModel.Spec.RemoteConfig.GetStatus(configHash)
			if currentStatus == remoteconfig.StatusApplied {
				// Agent already has this config applied, don't send config body again
				// According to OpAMP spec: "SHOULD NOT be set if the config for this Agent has not changed
				// since it was last requested (i.e. AgentConfigRequest.last_remote_config_hash field is equal
				// to AgentConfigResponse.config_hash field)."
				logger.Debug("agent already has the latest config applied",
					slog.String("agentGroupName", group.Name),
					slog.String("configHash", string(configHash)),
				)

				return &protobufs.AgentRemoteConfig{
					Config:     nil, // Don't send config body if agent already has it
					ConfigHash: configHash.Bytes(),
				}
			}

			logger.Info("sending remote config from agent group",
				slog.String("agentGroupName", group.Name),
				slog.String("configHash", string(configHash)),
				slog.String("currentStatus", currentStatus.String()),
			)

			// Build the AgentRemoteConfig message
			configMap := &protobufs.AgentConfigMap{
				ConfigMap: map[string]*protobufs.AgentConfigFile{
					"config": {
						Body: configBytes,
					},
				},
			}

			return &protobufs.AgentRemoteConfig{
				Config:     configMap,
				ConfigHash: configHash.Bytes(),
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
