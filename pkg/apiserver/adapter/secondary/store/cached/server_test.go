package cached_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clocktesting "k8s.io/utils/clock/testing"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/cached"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	domainport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
)

func TestServerStore_ClonesAndRefreshesStaleHeartbeat(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	now := time.Now()
	persistence := inmemory.NewServerRepository()
	passiveClock := clocktesting.NewFakeClock(now)
	cachedStore := cached.NewServerStore(persistence, passiveClock, time.Minute)
	t.Cleanup(cachedStore.Shutdown)

	var store domainport.Reader[string, *agentmodel.Server] = cachedStore

	server := &agentmodel.Server{ID: "peer", Address: "original", LastHeartbeatAt: now}
	require.NoError(t, persistence.PutServer(ctx, server))
	first, err := store.Get(ctx, server.ID)
	require.NoError(t, err)

	first.Address = "caller-mutation"
	second, err := store.Get(ctx, server.ID)
	require.NoError(t, err)
	assert.Equal(t, "original", second.Address)
	second.Address = "cached-read-mutation"
	// A cached expired heartbeat must not hide a newer persisted heartbeat.
	server.Address = "new-address"
	server.LastHeartbeatAt = now.Add(2 * time.Minute)
	require.NoError(t, persistence.PutServer(ctx, server))
	fresh, err := cachedStore.GetFresh(ctx, server.ID)
	require.NoError(t, err)
	assert.Equal(t, "new-address", fresh.Address)

	passiveClock.SetTime(server.LastHeartbeatAt)
	refreshed, err := store.Get(ctx, server.ID)
	require.NoError(t, err)
	assert.Equal(t, "new-address", refreshed.Address)
	assert.Equal(t, server.LastHeartbeatAt, refreshed.LastHeartbeatAt)
}

func TestServerStore_ShutdownClearsCache(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	now := time.Now()
	persistence := inmemory.NewServerRepository()
	store := cached.NewServerStore(persistence, clocktesting.NewFakeClock(now), time.Minute)
	t.Cleanup(store.Shutdown)

	server := &agentmodel.Server{ID: "peer", Address: "old", LastHeartbeatAt: now}
	require.NoError(t, persistence.PutServer(ctx, server))
	_, err := store.Get(ctx, server.ID)
	require.NoError(t, err)

	server.Address = "new"
	require.NoError(t, persistence.PutServer(ctx, server))
	store.Shutdown()
	got, err := store.Get(ctx, server.ID)
	require.NoError(t, err)
	assert.Equal(t, "new", got.Address)
}
