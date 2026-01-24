package opamp

import (
	"context"
	"errors"

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
func (s *Service) fetchServerToAgent(ctx context.Context, agentModel *model.Agent) *protobufs.ServerToAgent {
	var flags uint64

	instanceUID := agentModel.Metadata.InstanceUID

	var (
		remoteConfig *protobufs.AgentRemoteConfig
	)

	if agentModel.HasRemoteConfig() {
		// Build RemoteConfig if applicable
		remoteConfigs := lo.OmitBy(lo.SliceToMap(
			agentModel.Spec.RemoteConfig.RemoteConfig,
			func(remoteConfigName string) (string, *protobufs.AgentConfigFile) {
				remoteConfig, err := s.agentRemoteConfigUsecase.GetAgentRemoteConfig(ctx, remoteConfigName)
				if err != nil {
					s.logger.Error("failed to get agent remote config", "name", remoteConfigName, "error", err)
					return remoteConfigName, nil
				}
				return remoteConfigName, &protobufs.AgentConfigFile{
					Body:        remoteConfig.Value,
					ContentType: remoteConfig.ContentType,
				}
			}), nil)

		hash, err := vo.NewHashFromAny(remoteConfigs)
		if err != nil {
			s.logger.Error("failed to compute hash for remote config", "instance_uid", instanceUID, "error", err)
			remoteConfig = nil
		} else {
			remoteConfig = &protobufs.AgentRemoteConfig{
				Config: &protobufs.AgentConfigMap{
					ConfigMap: remoteConfigs,
				},
				ConfigHash: hash.Bytes(),
			}
		}
	}

	var agentIdentification *protobufs.AgentIdentification
	if agentModel.HasNewInstanceUID() {
		// Agent has a new InstanceUID, need to inform the agent
		agentIdentification = &protobufs.AgentIdentification{
			NewInstanceUid: agentModel.NewInstanceUID(),
		}
	}

	var command *protobufs.ServerToAgentCommand
	if agentModel.ShouldBeRestarted() {
		command = &protobufs.ServerToAgentCommand{
			Type: protobufs.CommandType_CommandType_Restart,
		}
	}

	var packagesAvailable *protobufs.PackageAvailable
	if len(agentModel.Spec.PackagesAvailable.Packages) > 0 {
		packagesAvailable = &protobufs.PackageAvailable{
			Packages: lo.Map(agentModel.Spec.PackagesAvailable.Packages, func(pkgName string, _ int) *protobufs.PackageInfo {
				return &protobufs.PackageInfo{
					Name: pkgName,
				}
			}),
			ConfigHash: agentModel.Spec.PackagesAvailable.Hash.Bytes(),
		}
	}

	var capabilities int32

	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsStatus)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_OffersRemoteConfig)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsEffectiveConfig)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsConnectionSettingsRequest)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_OffersConnectionSettings)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_OffersPackages)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsPackagesStatus)

	var connectionSettings *protobufs.ConnectionSettingsOffers
	if agentModel.Spec.ConnectionInfo.HasConnectionSettings() {
		connectionSettings = connectionInfoToProtobuf(&agentModel.Spec.ConnectionInfo)
	}

	return &protobufs.ServerToAgent{
		InstanceUid:         instanceUID[:],
		ErrorResponse:       nil,
		RemoteConfig:        remoteConfig,
		ConnectionSettings:  connectionSettings,
		PackagesAvailable:   nil,
		Flags:               flags,
		Capabilities:        uint64(capabilities), //nolint:gosec // safe conversion from int32 to uint64
		AgentIdentification: agentIdentification,
		Command:             command,
		CustomCapabilities:  nil,
		CustomMessage:       nil,
	}
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
