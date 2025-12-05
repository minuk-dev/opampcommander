package opamp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/vo"
)

var (
	// ErrNotSupportedOperation is returned when the operation is not supported by the agent.
	ErrNotSupportedOperation = errors.New("operation not supported by the agent")
)

// fetchServerToAgent creates a ServerToAgent message from the agent.
func (s *Service) fetchServerToAgent(ctx context.Context, agentModel *model.Agent) (*protobufs.ServerToAgent, error) {
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

	var (
		remoteConfig *protobufs.AgentRemoteConfig
		err          error
	)

	if agentModel.HasRemoteConfig() {
		// Build RemoteConfig if applicable
		remoteConfig, err = s.buildRemoteConfig(ctx, agentModel)
		if err != nil {
			s.logger.Error("failed to build remote config for agent",
				slog.String("instanceUID", instanceUID.String()),
				slog.String("error", err.Error()))

			return nil, fmt.Errorf("failed to build remote config: %w", err)
		}
	}

	var agentIdentification *protobufs.AgentIdentification
	if agentModel.HasNewInstanceUID() {
		// Agent has a new InstanceUID, need to inform the agent
		agentIdentification = &protobufs.AgentIdentification{
			NewInstanceUid: agentModel.NewInstanceUID(),
		}
	}

	var capabilities int32

	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsStatus)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_OffersRemoteConfig)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsEffectiveConfig)

	return &protobufs.ServerToAgent{
		InstanceUid:         instanceUID[:],
		ErrorResponse:       nil,
		RemoteConfig:        remoteConfig,
		ConnectionSettings:  nil,
		PackagesAvailable:   nil,
		Flags:               flags,
		Capabilities:        uint64(capabilities), //nolint:gosec // safe conversion from int32 to uint64
		AgentIdentification: agentIdentification,
		Command:             nil,
		CustomCapabilities:  nil,
		CustomMessage:       nil,
	}, nil
}

// buildRemoteConfig builds the remote configuration for the agent based on its agent groups.
func (s *Service) buildRemoteConfig(
	ctx context.Context,
	agentModel *model.Agent,
) (*protobufs.AgentRemoteConfig, error) {
	if !agentModel.IsRemoteConfigSupported() {
		return nil, ErrNotSupportedOperation
	}

	// Get agent groups for this agent
	agentGroups, err := s.agentGroupUsecase.GetAgentGroupsForAgent(ctx, agentModel)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent groups for agent: %w", err)
	}

	agentGroupsWithRemoteConfigs := lo.Filter(agentGroups, func(group *model.AgentGroup, _ int) bool {
		return group.Spec.AgentConfig != nil && group.Spec.AgentConfig.Value != ""
	})

	if len(agentGroupsWithRemoteConfigs) == 0 {
		//nolint:nilnil // Agent does not support remote config
		return nil, nil
	}

	sort.Slice(agentGroupsWithRemoteConfigs, func(i, j int) bool {
		return agentGroupsWithRemoteConfigs[i].Metadata.Priority > agentGroupsWithRemoteConfigs[j].Metadata.Priority
	})

	winnerGroup := agentGroupsWithRemoteConfigs[0]
	s.logger.Info("applying remote config from agent group",
		slog.String("agentGroupName", winnerGroup.Metadata.Name),
		slog.Int("priority", winnerGroup.Metadata.Priority),
	)

	configBytes := []byte(winnerGroup.Spec.AgentConfig.Value)

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
