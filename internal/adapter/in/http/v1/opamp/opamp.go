package opamp

import (
	"context"

	"github.com/open-telemetry/opamp-go/protobufs"
)

type opampHandler struct {
}

func newOpampHandler() *opampHandler {
	return &opampHandler{}
}

func (h *opampHandler) handleAgentToServer(context.Context, *protobufs.AgentToServer) error {
	return nil
}

func (h *opampHandler) fetchServerToAgent(context.Context) (*protobufs.ServerToAgent, error) {
	return nil, nil
}
