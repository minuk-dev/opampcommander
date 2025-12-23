package opamp

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

var (
	// ErrNotSupportedOperation is returned when the operation is not supported by the agent.
	ErrNotSupportedOperation = errors.New("operation not supported by the agent")
)

// fetchServerToAgent creates a ServerToAgent message from the agent.
func (s *Service) fetchServerToAgent(_ context.Context, agentModel *model.Agent) *protobufs.ServerToAgent {
	var flags uint64

	instanceUID := agentModel.Metadata.InstanceUID

	var (
		remoteConfig *protobufs.AgentRemoteConfig
	)

	if agentModel.HasRemoteConfig() {
		// Build RemoteConfig if applicable
		remoteConfig = &protobufs.AgentRemoteConfig{
			Config: &protobufs.AgentConfigMap{
				ConfigMap: map[string]*protobufs.AgentConfigFile{
					"opampcommander": {
						Body:        agentModel.Spec.RemoteConfig.Config,
						ContentType: agentModel.Spec.RemoteConfig.ContentType,
					},
				},
			},
			ConfigHash: agentModel.Spec.RemoteConfig.Hash.Bytes(),
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
