package usecase

import (
	"context"

	"github.com/open-telemetry/opamp-go/protobufs"
	opamptypes "github.com/open-telemetry/opamp-go/server/types"
)

// OpAMPUsecase is a use case that handles OpAMP protocol operations.
// Please see [github.com/open-telemetry/opamp-go/server/types/ConnectionCallbacks].
type OpAMPUsecase interface {
	OnConnected(ctx context.Context, conn opamptypes.Connection)
	OnConnectedWithType(ctx context.Context, conn opamptypes.Connection, isWebSocket bool)
	OnMessage(ctx context.Context, conn opamptypes.Connection, message *protobufs.AgentToServer) *protobufs.ServerToAgent
	OnConnectionClose(conn opamptypes.Connection)
	OnReadMessageError(conn opamptypes.Connection, mt int, msgByte []byte, err error)
	OnMessageResponseError(conn opamptypes.Connection, message *protobufs.ServerToAgent, err error)
}
