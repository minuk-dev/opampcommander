package agentport

import (
	"context"

	"github.com/google/uuid"
)

// PendingNotification contains the deduplicated agent updates for one server.
type PendingNotification struct {
	TargetServerID string
	InstanceUIDs   []uuid.UUID
}

// NotificationBatch is one queued inter-server notification.
type NotificationBatch struct {
	PendingNotification

	SourceServerID string
}

// NotificationStore owns pending UID sets, flush signals, and dispatch batches.
// It is a node-local, best-effort accelerator; pending agent messages themselves
// remain in durable agent storage. Drained values belong to the caller.
type NotificationStore interface {
	Enqueue(serverID string, uid uuid.UUID, maxBatchSize int)
	FlushSignal() <-chan struct{}
	DrainPending() []PendingNotification
	EnqueueBatch(ctx context.Context, batch NotificationBatch) error
	NextBatch(ctx context.Context) (NotificationBatch, error)
}
