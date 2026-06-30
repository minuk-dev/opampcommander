package usecase

import (
	"context"

	"github.com/open-telemetry/opamp-go/protobufs"
	opamptypes "github.com/open-telemetry/opamp-go/server/types"
)

// OpAMPUsecase handles the OpAMP protocol itself. Unlike the REST
// management use cases, it is implemented by the central OpAMP service and
// invoked by the opamp-go server adapter as connection callbacks for each
// connected agent.
// Please see [github.com/open-telemetry/opamp-go/server/types/ConnectionCallbacks].
type OpAMPUsecase interface {
	// OnConnected is called when an agent connection is established.
	OnConnected(ctx context.Context, conn opamptypes.Connection)
	// OnConnectedWithType is OnConnected with the transport kind (true for
	// WebSocket, false for plain HTTP) made explicit.
	OnConnectedWithType(ctx context.Context, conn opamptypes.Connection, isWebSocket bool)
	// OnMessage handles an AgentToServer message and returns the ServerToAgent
	// reply to send back over the same connection.
	OnMessage(ctx context.Context, conn opamptypes.Connection, message *protobufs.AgentToServer) *protobufs.ServerToAgent
	// OnConnectionClose releases the per-connection state when an agent
	// disconnects.
	OnConnectionClose(conn opamptypes.Connection)
	// OnReadMessageError reports a failure to read/parse an inbound frame.
	OnReadMessageError(conn opamptypes.Connection, mt int, msgByte []byte, err error)
	// OnMessageResponseError reports a failure to deliver a server reply.
	OnMessageResponseError(conn opamptypes.Connection, message *protobufs.ServerToAgent, err error)
}
