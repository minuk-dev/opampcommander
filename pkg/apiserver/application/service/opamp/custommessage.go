package opamp

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/open-telemetry/opamp-go/protobufs"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
)

// ErrDuplicateCustomCapabilityHandler is returned when two custom-message handlers claim the
// same custom capability, which would make dispatch ambiguous.
var ErrDuplicateCustomCapabilityHandler = errors.New("duplicate custom capability handler")

// CustomMessage is a vendor-specific OpAMP custom message, decoupled from the wire protobuf so
// handlers do not depend on opamp-go types. It mirrors [protobufs.CustomMessage]: Capability is
// the custom-capability string the message belongs to, Type is a capability-scoped message type,
// and Data is the opaque payload.
type CustomMessage struct {
	Capability string
	Type       string
	Data       []byte
}

// CustomMessageHandler handles inbound OpAMP custom messages for a single custom capability.
// Implementations plug into the dispatch seam via the FX "opampCustomMessageHandlers" group and
// are routed by capability string, so a new custom protocol can be added without touching the
// core OpAMP handler.
type CustomMessageHandler interface {
	// Capability returns the custom-capability string this handler serves, e.g.
	// "com.example.my-capability". The server advertises it in ServerToAgent.custom_capabilities
	// and only routes inbound messages carrying this capability to this handler.
	Capability() string
	// HandleCustomMessage processes one inbound custom message from the agent. It may return an
	// outbound custom message to send back in the same ServerToAgent response, or nil for no
	// reply. The reply is sent only if the agent has advertised the reply's capability.
	HandleCustomMessage(ctx context.Context, agent *agentmodel.Agent, msg *CustomMessage) (*CustomMessage, error)
}

// CustomMessageRegistry indexes the registered custom-message handlers by capability and exposes
// the set of capabilities the server advertises. An empty registry (no handlers registered) is
// the default and leaves custom-message exchange effectively disabled.
type CustomMessageRegistry struct {
	handlers     map[string]CustomMessageHandler
	capabilities agentservice.ServerCustomCapabilities
}

// NewCustomMessageRegistry indexes the given handlers by capability. It returns an error if two
// handlers claim the same capability, since that would make dispatch ambiguous — a wiring bug
// worth failing startup over rather than silently picking one.
func NewCustomMessageRegistry(handlers []CustomMessageHandler) (*CustomMessageRegistry, error) {
	byCapability := make(map[string]CustomMessageHandler, len(handlers))

	for _, handler := range handlers {
		capability := handler.Capability()
		if _, exists := byCapability[capability]; exists {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateCustomCapabilityHandler, capability)
		}

		byCapability[capability] = handler
	}

	// Sorted so the advertised capability list — and thus the ServerToAgent it feeds — is
	// deterministic regardless of handler registration order.
	capabilities := slices.Sorted(maps.Keys(byCapability))

	return &CustomMessageRegistry{
		handlers:     byCapability,
		capabilities: capabilities,
	}, nil
}

// Handler returns the handler registered for the given custom capability, if any.
//
//nolint:ireturn // A registry lookup necessarily returns the handler interface it stores.
func (r *CustomMessageRegistry) Handler(capability string) (CustomMessageHandler, bool) {
	handler, ok := r.handlers[capability]

	return handler, ok
}

// Capabilities returns the sorted set of custom capabilities the server advertises.
func (r *CustomMessageRegistry) Capabilities() agentservice.ServerCustomCapabilities {
	return r.capabilities
}

// customMessageToDomain converts a wire custom message into the handler-facing value type.
func customMessageToDomain(msg *protobufs.CustomMessage) *CustomMessage {
	if msg == nil {
		return nil
	}

	return &CustomMessage{
		Capability: msg.GetCapability(),
		Type:       msg.GetType(),
		Data:       msg.GetData(),
	}
}

// customMessageToProtobuf converts a handler-produced custom message into the wire type.
func customMessageToProtobuf(msg *CustomMessage) *protobufs.CustomMessage {
	if msg == nil {
		return nil
	}

	return &protobufs.CustomMessage{
		Capability: msg.Capability,
		Type:       msg.Type,
		Data:       msg.Data,
	}
}
