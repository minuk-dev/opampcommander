// Package command provides the command api for the opampcommander
package command

import "github.com/google/uuid"

// UpdateAgentConfigRequest is a struct that represents the request to update the agent configuration.
// It contains the target instance UID and the remote configuration data.
type UpdateAgentConfigRequest struct {
	TargetInstanceUID uuid.UUID `binding:"required" json:"targetInstanceUid"`
	RemoteConfig      any       `binding:"required" json:"remoteConfig"`
}
