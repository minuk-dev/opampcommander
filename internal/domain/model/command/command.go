package command

import (
	"github.com/google/uuid"
)

type UpdateAgentConfigCommand struct {
	// ID is a unique identifier for the command.
	// It is used to AgentConfigMap's key.
	ID uuid.UUID

	// TargetInstanceUID is a target instance to apply RemoteConfig.
	TargetInstanceUID uuid.UUID

	// RemoteConfig is a remote config to Apply.
	RemoteConfig any
}

func NewUpdateAgentConfigCommand(id uuid.UUID, targetInstanceUID uuid.UUID, remoteConfig any) UpdateAgentConfigCommand {
	return UpdateAgentConfigCommand{
		ID:                id,
		TargetInstanceUID: targetInstanceUID,
		RemoteConfig:      remoteConfig,
	}
}
