// Package port is a package that defines the ports for the application layer.
package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	opamptypes "github.com/open-telemetry/opamp-go/server/types"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// OpAMPUsecase is a use case that handles OpAMP protocol operations.
// Please see [github.com/open-telemetry/opamp-go/server/types/ConnectionCallbacks].
type OpAMPUsecase interface {
	OnConnected(ctx context.Context, conn opamptypes.Connection)
	OnMessage(ctx context.Context, conn opamptypes.Connection, message *protobufs.AgentToServer) *protobufs.ServerToAgent
	OnConnectionClose(conn opamptypes.Connection)
}

// AdminUsecase is a use case that handles admin operations.
type AdminUsecase interface {
	ApplyRawConfig(ctx context.Context, targetInstanceUID uuid.UUID, config any) error
	ListConnections(ctx context.Context) ([]*model.Connection, error)
}
