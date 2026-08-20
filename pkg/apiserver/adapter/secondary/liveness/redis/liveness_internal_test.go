package redis // simulating a TTL expiry means deleting the record key behind the store's back

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	redisTestContainer "github.com/testcontainers/testcontainers-go/modules/redis"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

func TestStorePendingSelfHealsAnOrphanedIndexEntry(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping redis container test in short mode")
	}

	// The container is started directly rather than through pkg/testutil: that
	// package pulls in the whole apiserver, which imports this one.
	container, err := redisTestContainer.Run(t.Context(), "redis:7.4-alpine")
	require.NoError(t, err)

	endpoint, err := container.Endpoint(t.Context(), "")
	require.NoError(t, err)

	//exhaustruct:ignore
	store, err := New(Config{
		Endpoints:      []string{endpoint},
		DialTimeout:    2 * time.Second,
		CommandTimeout: 2 * time.Second,
		TTL:            2 * time.Minute,
	})
	require.NoError(t, err)

	t.Cleanup(func() { _ = store.Close() })

	instanceUID := uuid.New()
	now := time.Now()

	observation := &agentmodel.AgentLiveness{
		InstanceUID:       instanceUID,
		Connected:         true,
		ConnectionType:    agentmodel.ConnectionTypeWebSocket,
		SequenceNum:       1,
		LastReportedAt:    now,
		LastReportedTo:    "server-a",
		DurableReportedAt: time.Time{},
	}

	_, err = store.Touch(t.Context(), observation)
	require.NoError(t, err)
	require.NoError(t, store.MarkPersisted(t.Context(), instanceUID, now.Add(-10*time.Minute)))

	_, err = store.Touch(t.Context(), observation)
	require.NoError(t, err)

	// Drop the record the way a TTL expiry would, leaving the index entry behind.
	require.NoError(t, store.client.Del(t.Context(), store.recordKey(instanceUID)).Err())

	pending, err := store.ListPendingWriteThrough(t.Context(), now, 0)
	require.NoError(t, err)
	assert.Empty(t, pending)

	// The orphaned entry must be swept, or the index would grow without bound as
	// agents disappear — nothing else removes them.
	size, err := store.client.ZCard(t.Context(), store.pendingKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)
}
