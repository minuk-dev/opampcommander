// Package serverevent defines server-to-server event models.
package serverevent

import "github.com/google/uuid"

// MessageType represents a message sent to a server.
type MessageType string

// String returns the string representation of the ServerMessageType.
func (s MessageType) String() string {
	return string(s)
}

const (
	// MessageTypeSendServerToAgent is a message type for sending ServerToAgent messages for specific agents.
	MessageTypeSendServerToAgent MessageType = "SendServerToAgent"
	// MessageTypeInvalidateAgentCache asks the recipient server to drop its cached copy of
	// the listed agents, so a write made on another node is not served stale from cache.
	MessageTypeInvalidateAgentCache MessageType = "InvalidateAgentCache"
)

// Message represents a message sent between servers.
type Message struct {
	// Source is the identifier of the message sender of server.
	Source string
	// Target is the identifier of the message recipient agent.
	Target string
	// Type is the type of the message.
	Type MessageType
	// Payload is the payload of the message.
	Payload MessagePayload
}

// MessagePayload represents the payload of a server event message.
type MessagePayload struct {
	// When Type is ServerMessageTypeSendServerToAgent, Payload is ServerToAgentMessage
	*MessageForServerToAgent
	// When Type is MessageTypeInvalidateAgentCache, Payload is MessageForInvalidateAgentCache.
	*MessageForInvalidateAgentCache
}

// MessageForServerToAgent represents a message sent from the server to an agent.
// It's encoded as json in the CloudEvent data field.
type MessageForServerToAgent struct {
	// TargetAgentInstanceUIDs is the list of target agent instance UIDs.
	// Do not send details message, the target server should fetch the details from the database
	// because the message can be delayed or missed.
	// All servers should check all agents status periodically to handle such cases.
	TargetAgentInstanceUIDs []uuid.UUID `json:"targetAgentInstanceUids"`
}

// MessageForInvalidateAgentCache carries the agents whose cached copies the recipient
// server should drop. It's encoded as json in the CloudEvent data field.
type MessageForInvalidateAgentCache struct {
	// AgentInstanceUIDs is the list of agent instance UIDs to invalidate from the cache.
	AgentInstanceUIDs []uuid.UUID `json:"agentInstanceUids"`
}
