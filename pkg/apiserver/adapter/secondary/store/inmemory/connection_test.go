package inmemory_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

func TestConnectionStoreOwnsStoredModels(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	store := inmemory.NewConnectionStore()
	uid := uuid.New()
	connection := agentmodel.NewConnection(new(int), agentmodel.ConnectionTypeWebSocket)
	connection.SetInstanceUID(uid)
	require.NoError(t, store.PutConnection(ctx, connection))
	connection.Namespace = "changed-without-saving"
	stored, err := store.GetConnection(ctx, connection.ID)
	require.NoError(t, err)
	assert.Equal(t, "default", stored.Namespace)
	stored.Namespace = "changed-after-reading"
	listed, err := store.ListConnections(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "default", listed[0].Namespace)
	listed[0].Namespace = "changed-after-listing"
	indexed, err := store.GetConnectionByInstanceUID(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, "default", indexed.Namespace)

	otherUID := uuid.New()
	indexed.SetInstanceUID(otherUID)
	require.NoError(t, store.PutConnection(ctx, indexed))
	_, err = store.GetConnectionByInstanceUID(ctx, uid)
	require.ErrorIs(t, err, agentport.ErrConnectionNotFound)
	indexed, err = store.GetConnectionByInstanceUID(ctx, otherUID)
	require.NoError(t, err)
	assert.Equal(t, otherUID, indexed.InstanceUID)
}

func TestConnectionStoreDeletionPreservesReplacement(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		reuseID bool
	}{
		{name: "replacement uses another transport"},
		{name: "replacement reuses transport ID", reuseID: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			store := inmemory.NewConnectionStore()
			uid := uuid.New()
			old := agentmodel.NewConnection(new(int), agentmodel.ConnectionTypeWebSocket)
			old.SetInstanceUID(uid)

			current := agentmodel.NewConnection(new(int), agentmodel.ConnectionTypeWebSocket)
			current.SetInstanceUID(uid)

			if tc.reuseID {
				current.ID = old.ID
			}

			require.NoError(t, store.PutConnection(ctx, old))
			require.NoError(t, store.PutConnection(ctx, current))
			require.NoError(t, store.DeleteConnection(ctx, old))
			active, err := store.GetConnectionByInstanceUID(ctx, uid)
			require.NoError(t, err)
			assert.Equal(t, current.UID, active.UID)
			active, err = store.GetConnection(ctx, current.ID)
			require.NoError(t, err)
			assert.Equal(t, current.UID, active.UID)
		})
	}
}

func TestConnectionStoreSessionErrors(t *testing.T) {
	t.Parallel()

	store := inmemory.NewConnectionStore()
	uid := uuid.New()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	called := false
	err := store.WithinSession(ctx, uid, func(context.Context) error {
		called = true

		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, called)
	err = store.WithinSession(t.Context(), uid, func(context.Context) error {
		return context.DeadlineExceeded
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	err = store.WithinSession(t.Context(), uid, func(context.Context) error {
		called = true

		return nil
	})
	require.NoError(t, err)
	assert.True(t, called, "canceling or failing must release the session scope")
}

func TestConnectionStoreClosedQueue(t *testing.T) {
	t.Parallel()

	store := inmemory.NewConnectionStore()
	id := new(int)
	require.True(t, store.EnqueueClosedConnection(id))
	assert.False(t, store.EnqueueClosedConnection(new(int)), "a full queue must not block the caller")
	closed, err := store.NextClosedConnection(t.Context())
	require.NoError(t, err)
	assert.Same(t, id, closed)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = store.NextClosedConnection(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestConnectionStoreOwnsSnapshotState(t *testing.T) {
	t.Parallel()

	store := inmemory.NewConnectionStore()
	uid := uuid.New()
	state := agentport.ConnectionSnapshotState{
		Reconciled: true,
		Connections: map[uuid.UUID]agentmodel.ServerConnection{
			uid: {UID: uid, Namespace: "default"},
		},
	}
	store.SaveSnapshotState(state)
	delete(state.Connections, uid)

	stored := store.SnapshotState()
	assert.True(t, stored.Reconciled)
	require.Contains(t, stored.Connections, uid)
	delete(stored.Connections, uid)
	assert.Contains(t, store.SnapshotState().Connections, uid)
}
