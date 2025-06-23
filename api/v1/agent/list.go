package agent

import "github.com/google/uuid"

// Agent represents an agent which is defined OpAMP protocol.
// It is a value object that contains the instance UID and raw data.
type Agent struct {
	// InstanceUID is a unique identifier for the agent instance.
	InstanceUID uuid.UUID `json:"instanceUid"`

	// Raw is a raw data of the agent.
	// It is used for debugging purposes.
	Raw any `json:"raw"`
} // @name Agent
