package inmemory_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

func TestNotificationStore_DeduplicatesTargetsAndSignalsFlush(t *testing.T) {
	t.Parallel()

	store := inmemory.NewNotificationStore(2)
	first, second := uuid.New(), uuid.New()
	store.Enqueue("a", first, 2)
	store.Enqueue("a", first, 2)

	select {
	case <-store.FlushSignal():
		t.Fatal("duplicate UID triggered flush")
	default:
	}

	store.Enqueue("a", second, 2)
	store.Enqueue("b", first, 2)

	select {
	case <-store.FlushSignal():
	default:
		t.Fatal("batch limit did not trigger flush")
	}

	drained := store.DrainPending()
	require.Len(t, drained, 2)

	byTarget := make(map[string][]uuid.UUID)
	for _, pending := range drained {
		byTarget[pending.TargetServerID] = pending.InstanceUIDs
	}

	assert.ElementsMatch(t, []uuid.UUID{first, second}, byTarget["a"])
	assert.Equal(t, []uuid.UUID{first}, byTarget["b"])
	assert.Empty(t, store.DrainPending())
	store.Enqueue("a", first, 2)
	require.Len(t, store.DrainPending(), 1)
}

func TestNotificationStore_ConcurrentEnqueueAndDrain(t *testing.T) {
	t.Parallel()

	store := inmemory.NewNotificationStore(1)

	const count = 500

	uids := make([]uuid.UUID, count)
	for i := range uids {
		uids[i] = uuid.New()
	}

	var workers sync.WaitGroup
	for worker := range 4 {
		workers.Go(func() {
			for i := worker; i < count; i += 4 {
				store.Enqueue("target", uids[i], count)
			}
		})
	}

	done := make(chan struct{})

	go func() { workers.Wait(); close(done) }()

	var collected []uuid.UUID

	for {
		for _, pending := range store.DrainPending() {
			collected = append(collected, pending.InstanceUIDs...)
		}

		select {
		case <-done:
			for _, pending := range store.DrainPending() {
				collected = append(collected, pending.InstanceUIDs...)
			}

			assert.ElementsMatch(t, uids, collected)

			return
		default:
		}
	}
}

func TestNotificationStore_BatchIsolationAndCancellation(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	store := inmemory.NewNotificationStore(1)
	uid := uuid.New()
	batch := agentport.NotificationBatch{SourceServerID: "source", PendingNotification: agentport.PendingNotification{
		TargetServerID: "target", InstanceUIDs: []uuid.UUID{uid},
	}}
	require.NoError(t, store.EnqueueBatch(ctx, batch))
	batch.InstanceUIDs[0] = uuid.New()
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	// The full dispatch queue must unblock producers on cancellation.
	require.ErrorIs(t, store.EnqueueBatch(cancelled, batch), context.Canceled)
	got, err := store.NextBatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, uid, got.InstanceUIDs[0])
	assert.Equal(t, "source", got.SourceServerID)

	_, err = store.NextBatch(cancelled)
	require.ErrorIs(t, err, context.Canceled)
}
