package inmemory

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/google/uuid"

	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/datastructure/sets"
)

var _ agentport.NotificationStore = (*NotificationStore)(nil)

type pendingBucket struct {
	mu   sync.Mutex
	uids sets.UUID
}

// NotificationStore coalesces updates per target and owns the dispatch queue.
// Bucket locks allow writes to different targets without a shared outer lock.
type NotificationStore struct {
	pending  sync.Map
	flushNow chan struct{}
	dispatch chan agentport.NotificationBatch
}

// NewNotificationStore creates a bounded, node-local notification store.
func NewNotificationStore(queueSize int) *NotificationStore {
	return &NotificationStore{
		pending:  sync.Map{},
		flushNow: make(chan struct{}, 1),
		dispatch: make(chan agentport.NotificationBatch, queueSize),
	}
}

// Enqueue deduplicates a UID and requests a flush when its target reaches the limit.
func (s *NotificationStore) Enqueue(serverID string, uid uuid.UUID, maxBatchSize int) {
	value, found := s.pending.Load(serverID)
	if !found {
		value, _ = s.pending.LoadOrStore(serverID, &pendingBucket{mu: sync.Mutex{}, uids: sets.NewUUID()})
	}

	bucket, ok := value.(*pendingBucket)
	if !ok {
		return
	}

	bucket.mu.Lock()
	bucket.uids.Insert(uid)
	size := bucket.uids.Len()
	bucket.mu.Unlock()

	if size >= maxBatchSize {
		select {
		case s.flushNow <- struct{}{}:
		default:
		}
	}
}

// FlushSignal exposes only the read end of the coalesced wake-up signal.
func (s *NotificationStore) FlushSignal() <-chan struct{} { return s.flushNow }

// DrainPending atomically detaches each target's pending UID set. Enqueues after
// a bucket is drained remain pending for the next flush.
func (s *NotificationStore) DrainPending() []agentport.PendingNotification {
	var drained []agentport.PendingNotification

	s.pending.Range(func(key, value any) bool {
		serverID, keyOK := key.(string)

		bucket, valueOK := value.(*pendingBucket)
		if !keyOK || !valueOK {
			return true
		}

		bucket.mu.Lock()
		if bucket.uids.Len() > 0 {
			drained = append(drained, agentport.PendingNotification{TargetServerID: serverID, InstanceUIDs: bucket.uids.List()})
			bucket.uids = sets.NewUUID()
		}
		bucket.mu.Unlock()

		return true
	})

	return drained
}

// EnqueueBatch stores an isolated batch, applying backpressure until cancelled.
func (s *NotificationStore) EnqueueBatch(ctx context.Context, batch agentport.NotificationBatch) error {
	batch.InstanceUIDs = slices.Clone(batch.InstanceUIDs)
	select {
	case <-ctx.Done():
		return fmt.Errorf("enqueue notification batch: %w", ctx.Err())
	case s.dispatch <- batch:
		return nil
	}
}

// NextBatch waits for a queued batch or cancellation. The caller owns the result.
func (s *NotificationStore) NextBatch(ctx context.Context) (agentport.NotificationBatch, error) {
	err := ctx.Err()
	if err != nil {
		return agentport.NotificationBatch{}, fmt.Errorf("next notification batch: %w", err)
	}

	select {
	case <-ctx.Done():
		return agentport.NotificationBatch{}, fmt.Errorf("next notification batch: %w", ctx.Err())
	case batch := <-s.dispatch:
		return batch, nil
	}
}
