package opamp

import (
	"context"
	"errors"

	"github.com/open-telemetry/opamp-go/protobufs"
)

var ErrNotImplemented = errors.New("not implemented")

type opampHandler struct{}

func newOpampHandler() *opampHandler {
	return &opampHandler{}
}

func (h *opampHandler) handleAgentToServer(context.Context, *protobufs.AgentToServer) error {
	return ErrNotImplemented
}

func (h *opampHandler) fetchServerToAgent(context.Context) (*protobufs.ServerToAgent, error) {
	return nil, ErrNotImplemented
}
