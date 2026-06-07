package common

import (
	"log/slog"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/inmemory"
)

// newInMemoryEventHub provides the shared in-memory event hub used in standalone
// (single-node) mode. A single instance backs both the outbound sender adapter
// (secondary) and the inbound receiver adapter (primary), so an event sent on one
// side is observed on the other.
//
// It is a cheap, side-effect-free value (a buffered channel), so it is harmless to
// construct even when Kafka is the active transport.
func newInMemoryEventHub(logger *slog.Logger) *inmemory.EventSenderAdapter {
	return inmemory.NewEventHubAdapter(logger)
}
