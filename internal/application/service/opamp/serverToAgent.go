package opamp

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
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
	remoteConfig, err := s.buildRemoteConfig(ctx, agentModel)
	if err != nil {
		s.logger.Error("failed to build remote config for agent",
			slog.String("instanceUID", instanceUID.String()),
			slog.String("error", err.Error()))

		return s.createFallbackServerToAgent(instanceUID)
	}

	//exhaustruct:ignore
	return &protobufs.ServerToAgent{
		InstanceUid:  instanceUID[:],
		Flags:        flags,
		RemoteConfig: remoteConfig,
	}
}

// buildRemoteConfig builds the remote configuration for the agent based on its agent groups.
func (s *Service) buildRemoteConfig(
	ctx context.Context,
	agentModel *model.Agent,
) (*protobufs.AgentRemoteConfig, error) {
	// Check if agent supports RemoteConfig
	if !agentModel.Metadata.Capabilities.Has(agent.AgentCapabilityAcceptsRemoteConfig) {
		//nolint:nilnil // Agent does not support remote config
		return nil, nil
	}

	// Get agent groups for this agent
	agentGroups, err := s.agentGroupUsecase.GetAgentGroupsForAgent(ctx, agentModel)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent groups for agent: %w", err)
	}

	agentGroupsWithRemoteConfigs := lo.Filter(agentGroups, func(group *agentgroup.AgentGroup, _ int) bool {
		return group.AgentConfig != nil && group.AgentConfig.Value != ""
	})

	if len(agentGroupsWithRemoteConfigs) == 0 {
		//nolint:nilnil // Agent does not support remote config
		return nil, nil
	}

	sort.Slice(agentGroupsWithRemoteConfigs, func(i, j int) bool {
		return agentGroupsWithRemoteConfigs[i].Priority > agentGroupsWithRemoteConfigs[j].Priority
	})

	winnerGroup := agentGroupsWithRemoteConfigs[0]
	s.logger.Info("applying remote config from agent group",
		slog.String("agentGroupName", winnerGroup.Name),
		slog.Int("priority", winnerGroup.Priority),
	)

	configBytes := []byte(winnerGroup.AgentConfig.Value)

	// Compute hash of the configuration
	configHash, err := vo.NewHash(configBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to compute config hash: %w", err)
	}

	// Check if agent already has this config applied
	currentStatus := agentModel.Spec.RemoteConfig.GetStatus(configHash)
	if currentStatus == model.RemoteConfigStatusApplied {
		// Agent already has this config applied, don't send config body again
		// According to OpAMP spec: "SHOULD NOT be set if the config for this Agent has not changed
		// since it was last requested (i.e. AgentConfigRequest.last_remote_config_hash field is equal
		// to AgentConfigResponse.config_hash field)."
		return &protobufs.AgentRemoteConfig{
			Config:     nil, // Don't send config body if agent already has it
			ConfigHash: configHash.Bytes(),
		}, nil
	}

	return &protobufs.AgentRemoteConfig{
		Config: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{
				"opampcommander": {
					Body: configBytes,
				},
			},
		},
		ConfigHash: configHash.Bytes(),
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
