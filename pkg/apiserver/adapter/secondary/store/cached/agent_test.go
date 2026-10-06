package cached_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/cached"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	domainport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
)

func TestAgentStore_CacheIsolationAndFreshRead(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	persistence := inmemory.NewAgentRepository()
	store := cached.NewAgentStore(persistence, cached.DefaultAgentCacheConfig())
	t.Cleanup(store.Shutdown)

	agent := agentmodel.NewAgent(uuid.New())
	require.NoError(t, store.Put(ctx, agent))
	version := agent.Metadata.ResourceVersion
	// Neither the saved model nor a returned cached model may mutate stored values.
	agent.Metadata.Namespace = "caller-mutation"
	first, err := store.Get(ctx, agent.Metadata.InstanceUID)
	require.NoError(t, err)
	assert.Equal(t, "default", first.Metadata.Namespace)
	first.Metadata.Namespace = "read-mutation"
	second, err := store.Get(ctx, agent.Metadata.InstanceUID)
	require.NoError(t, err)
	assert.Equal(t, "default", second.Metadata.Namespace)
	assert.Equal(t, version, second.Metadata.ResourceVersion)

	// A peer write stays invisible to cached reads until invalidation, but fresh
	// reads used by the deletion guard must see it immediately.
	peer, err := persistence.GetAgent(ctx, agent.Metadata.InstanceUID)
	require.NoError(t, err)

	peer.Metadata.Namespace = "peer-write"
	require.NoError(t, persistence.PutAgent(ctx, peer))
	fresh, err := store.GetFresh(ctx, peer.Metadata.InstanceUID)
	require.NoError(t, err)
	assert.Equal(t, "peer-write", fresh.Metadata.Namespace)
	stale, err := store.Get(ctx, peer.Metadata.InstanceUID)
	require.NoError(t, err)
	assert.Equal(t, "default", stale.Metadata.Namespace)
	store.InvalidateCache(peer.Metadata.InstanceUID)
	refreshed, err := store.Get(ctx, peer.Metadata.InstanceUID)
	require.NoError(t, err)
	assert.Equal(t, "peer-write", refreshed.Metadata.Namespace)
}

func TestAgentStore_ConflictInvalidatesAndDeleteRevokes(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	persistence := inmemory.NewAgentRepository()
	cachedStore := cached.NewAgentStore(persistence, cached.DefaultAgentCacheConfig())
	t.Cleanup(cachedStore.Shutdown)

	var store domainport.Store[uuid.UUID, *agentmodel.Agent] = cachedStore

	agent := agentmodel.NewAgent(uuid.New())
	require.NoError(t, store.Put(ctx, agent))
	loser, err := store.Get(ctx, agent.Metadata.InstanceUID)
	require.NoError(t, err)

	winner := loser.Clone()
	winner.Metadata.Namespace = "winner"
	require.NoError(t, persistence.PutAgent(ctx, winner))
	require.ErrorIs(t, store.Put(ctx, loser), model.ErrConflict)
	refreshed, err := store.Get(ctx, agent.Metadata.InstanceUID)
	require.NoError(t, err)
	assert.Equal(t, "winner", refreshed.Metadata.Namespace)
	require.NoError(t, store.Delete(ctx, agent.Metadata.InstanceUID))
	_, err = store.Get(ctx, agent.Metadata.InstanceUID)
	require.ErrorIs(t, err, model.ErrAgentRevoked)
}

func TestAgentStore_DisabledCacheAndShutdown(t *testing.T) {
	t.Parallel()

	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "shutdown"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			persistence := inmemory.NewAgentRepository()
			config := cached.DefaultAgentCacheConfig()
			config.Enabled = enabled
			store := cached.NewAgentStore(persistence, config)
			t.Cleanup(store.Shutdown)

			agent := agentmodel.NewAgent(uuid.New())
			require.NoError(t, store.Put(ctx, agent))
			peer := agent.Clone()
			peer.Metadata.Namespace = "peer"
			require.NoError(t, persistence.PutAgent(ctx, peer))

			if enabled {
				store.Shutdown()
			}

			got, err := store.Get(ctx, agent.Metadata.InstanceUID)
			require.NoError(t, err)
			assert.Equal(t, "peer", got.Metadata.Namespace)
		})
	}
}

func TestAgentStore_TTLExpires(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()
		persistence := inmemory.NewAgentRepository()
		config := cached.DefaultAgentCacheConfig()
		config.TTL = time.Second

		store := cached.NewAgentStore(persistence, config)
		defer store.Shutdown()

		agent := agentmodel.NewAgent(uuid.New())
		require.NoError(t, store.Put(ctx, agent))
		peer := agent.Clone()
		peer.Metadata.Namespace = "peer"
		require.NoError(t, persistence.PutAgent(ctx, peer))
		time.Sleep(2 * time.Second)

		got, err := store.Get(ctx, peer.Metadata.InstanceUID)
		require.NoError(t, err)
		assert.Equal(t, "peer", got.Metadata.Namespace)
	})
}
