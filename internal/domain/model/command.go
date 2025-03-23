package model

import (
	"github.com/google/uuid"
)

type CommandKind string

const (
	UpdateAgentConfigCommandKind CommandKind = "UpdateAgentConfig"
)

type Command struct {
	Kind CommandKind

	ID                uuid.UUID
	TargetInstanceUID uuid.UUID
	Data              map[string]any
}

func NewUpdateAgentConfigCommand(id uuid.UUID, targetInstanceUID uuid.UUID, remoteConfig any) *Command {
	return &Command{
		ID:                id,
		TargetInstanceUID: targetInstanceUID,
		Kind:              UpdateAgentConfigCommandKind,
		Data: map[string]any{
			"remoteConfig": remoteConfig,
		},
	}
}
