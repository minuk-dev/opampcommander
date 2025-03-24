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
