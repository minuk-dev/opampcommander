package model

import (
	"github.com/google/uuid"
)

// CommandKind represents the type of command to be sent to an agent.
type CommandKind string

const (
	// UpdateAgentConfigCommandKind is the command kind for updating agent configuration.
	UpdateAgentConfigCommandKind CommandKind = "UpdateAgentConfig"
)

// Command represents a command to be sent to an agent.
type Command struct {
	Kind CommandKind

	ID                uuid.UUID
	TargetInstanceUID uuid.UUID
	Data              map[string]any
}

// NewUpdateAgentConfigCommand creates a new command to update the agent configuration.
func NewUpdateAgentConfigCommand(targetInstanceUID uuid.UUID, remoteConfig any) *Command {
	return &Command{
		ID:                uuid.New(),
		TargetInstanceUID: targetInstanceUID,
		Kind:              UpdateAgentConfigCommandKind,
		Data: map[string]any{
			"remoteConfig": remoteConfig,
		},
	}
}
