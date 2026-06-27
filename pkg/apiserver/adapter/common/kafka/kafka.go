// Package kafka provides Kafka messaging models.
package kafka

import "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"

const (
	// CloudEventMessageSpec is the CloudEvents spec version used.
	CloudEventMessageSpec = "1.0"
	// CloudEventContentType is the CloudEvents content type used.
	CloudEventContentType = "application/json"
)

const (
	// SendToAgentEventType is the CloudEvent type for sending messages to agents.
	SendToAgentEventType = "io.opampcommander.server.sendtosagent.v1"
	// InvalidateAgentCacheEventType is the CloudEvent type for invalidating cached agents.
	InvalidateAgentCacheEventType = "io.opampcommander.server.invalidateagentcache.v1"
	// UnknownEventType is the CloudEvent type for unknown messages.
	UnknownEventType = "io.opampcommander.server.unknown.v1"
)

// EventTypeFromMessageType maps serverevent.MessageType to CloudEvent type string.
func EventTypeFromMessageType(messageType serverevent.MessageType) string {
	switch messageType {
	case serverevent.MessageTypeSendServerToAgent:
		return SendToAgentEventType
	case serverevent.MessageTypeInvalidateAgentCache:
		return InvalidateAgentCacheEventType
	default:
		return UnknownEventType
	}
}

// MessageTypeFromEventType maps CloudEvent type string to serverevent.MessageType.
func MessageTypeFromEventType(eventType string) (serverevent.MessageType, error) {
	switch eventType {
	case SendToAgentEventType:
		return serverevent.MessageTypeSendServerToAgent, nil
	case InvalidateAgentCacheEventType:
		return serverevent.MessageTypeInvalidateAgentCache, nil
	default:
		return "", &UnknownMessageTypeError{MessageType: eventType}
	}
}
