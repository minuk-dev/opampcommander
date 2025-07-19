// Package agent provides the agent API for the server
package agent

import "github.com/google/uuid"

const (
	// AgentKind is the kind of the agent resource.
	AgentKind = "Agent"
)

// UpdateAgentConfigRequest is a struct that represents the request to update the agent configuration.
// It contains the target instance UID and the remote configuration data.
type UpdateAgentConfigRequest struct {
	RemoteConfig any `binding:"required" json:"remoteConfig"`
} // @name UpdateAgentConfigRequest

// Agent represents an agent which is defined OpAMP protocol.
// It is a value object that contains the instance UID and raw data.
type Agent struct {
	// InstanceUID is a unique identifier for the agent instance.
	InstanceUID uuid.UUID `json:"instanceUid"`

	// Raw is a raw data of the agent.
	// It is used for debugging purposes.
	Raw any `json:"raw"`
} // @name Agent
