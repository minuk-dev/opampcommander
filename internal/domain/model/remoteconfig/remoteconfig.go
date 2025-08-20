// Package remoteconfig provides the remote config for opampcommander.
package remoteconfig

import (
	"time"

	"github.com/google/uuid"
)

type RemoteConfig struct {
	ID             uuid.UUID
	TargetSelector TargetSelector

	CreatedAt      time.Time
	CreatedBy      string
	LastModifiedAt time.Time
	LastModifiedBy string
}

type TargetSelector struct {
}

type InstanceUIDTargetSelector struct {
	InstanceUID uuid.UUID
}

type IdentifyingAttributesTargetSelector struct {
	IdentifyingAttributes map[string]string
}

type PredefinedTargetSelector struct {
	Target PredefinedTarget
}

type PredefinedTarget string

const (
	PredefinedTargetAllAgents PredefinedTarget = "ALL"
)
