package agentport

import (
	"context"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// ConnectionSnapshotState tracks this process's last successful cluster snapshot.
// It is storage bookkeeping, not agent state reported through the API.
type ConnectionSnapshotState struct {
	Reconciled  bool
	Connections map[uuid.UUID]agentmodel.ServerConnection
}

// ConnectionStore owns this server's live connection state, indexes, pending
// closes, and snapshot baseline. Returned models are copies; their opaque ID
// still identifies the original transport connection.
type ConnectionStore interface {
	GetConnection(ctx context.Context, id any) (*agentmodel.Connection, error)
	GetConnectionByInstanceUID(ctx context.Context, uid uuid.UUID) (*agentmodel.Connection, error)
	ListConnections(ctx context.Context) ([]*agentmodel.Connection, error)
	PutConnection(ctx context.Context, connection *agentmodel.Connection) error
	DeleteConnection(ctx context.Context, connection *agentmodel.Connection) error

	// WithinSession serializes work for one agent on this server, including
	// connection checks and associated agent/liveness writes. The callback may
	// call Store methods. This provides ordering, not rollback or an HA lock.
	WithinSession(ctx context.Context, uid uuid.UUID, fn func(context.Context) error) error

	EnqueueClosedConnection(id any) bool
	NextClosedConnection(ctx context.Context) (any, error)
	SnapshotState() ConnectionSnapshotState
	SaveSnapshotState(state ConnectionSnapshotState)
}
