// Package direct provides the shared wire encoding for the direct (peer-to-peer)
// server-event transport. The HTTP and gRPC adapters both build on these helpers
// so the two sub-protocols carry an identical payload.
package direct

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

// DeliverPath is the HTTP path the direct receiver serves and the sender POSTs to.
const DeliverPath = "/internal/v1/serverevents:deliver"

const (
	// AuthHeader is the HTTP header carrying the pre-shared bearer credential.
	AuthHeader = "Authorization"
	// AuthMetadataKey is the gRPC metadata key carrying the pre-shared bearer credential.
	AuthMetadataKey = "authorization"
	// BearerPrefix prefixes the token in both the HTTP header and the gRPC metadata.
	BearerPrefix = "Bearer "
)

// ConstantTimeTokenMatch reports whether presented equals expected without leaking timing.
// BearerPrefix is stripped from presented before comparison.
func ConstantTimeTokenMatch(expected, presented string) bool {
	presented = strings.TrimPrefix(presented, BearerPrefix)

	return subtle.ConstantTimeCompare([]byte(expected), []byte(presented)) == 1
}

// ErrUnknownMessageType is returned when an envelope carries a message type this
// server does not recognize.
var ErrUnknownMessageType = errors.New("unknown server event message type")

// Envelope is the JSON body exchanged by the HTTP direct transport. It mirrors the
// fields of serverevent.Message so a message round-trips without loss.
type Envelope struct {
	Source  string                     `json:"source"`
	Target  string                     `json:"target"`
	Type    string                     `json:"type"`
	Payload serverevent.MessagePayload `json:"payload"`
}

// ToEnvelope converts a domain message into its wire envelope.
func ToEnvelope(msg serverevent.Message) Envelope {
	return Envelope{
		Source:  msg.Source,
		Target:  msg.Target,
		Type:    msg.Type.String(),
		Payload: msg.Payload,
	}
}

// ToMessage converts a wire envelope back into a domain message, validating the type.
func (e Envelope) ToMessage() (serverevent.Message, error) {
	msgType, err := ParseMessageType(e.Type)
	if err != nil {
		return serverevent.Message{}, err
	}

	return serverevent.Message{
		Source:  e.Source,
		Target:  e.Target,
		Type:    msgType,
		Payload: e.Payload,
	}, nil
}

// ParseMessageType validates and converts a raw type string into a MessageType.
func ParseMessageType(raw string) (serverevent.MessageType, error) {
	switch serverevent.MessageType(raw) {
	case serverevent.MessageTypeSendServerToAgent, serverevent.MessageTypeInvalidateAgentCache:
		return serverevent.MessageType(raw), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownMessageType, raw)
	}
}

// EncodePayload serializes a message payload for the gRPC transport's bytes field.
func EncodePayload(payload serverevent.MessagePayload) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message payload: %w", err)
	}

	return data, nil
}

// DecodePayload deserializes a message payload from the gRPC transport's bytes field.
func DecodePayload(data []byte) (serverevent.MessagePayload, error) {
	var payload serverevent.MessagePayload

	err := json.Unmarshal(data, &payload)
	if err != nil {
		return serverevent.MessagePayload{}, fmt.Errorf("failed to decode message payload: %w", err)
	}

	return payload, nil
}
