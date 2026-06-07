package common

import (
	"log/slog"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/inmemory"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
)

// newInMemorySenderAndReceiver creates the in-memory event hub used in single-node
// (standalone) mode. A single hub instance backs both the sender and receiver ports.
func newInMemorySenderAndReceiver(
	logger *slog.Logger,
) (agentport.ServerEventSenderPort, agentport.ServerEventReceiverPort, error) {
	adapter := inmemory.NewEventHubAdapter(logger)

	return adapter, adapter, nil
}
