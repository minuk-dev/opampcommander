package mongodb_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestServerConnectionMongoAdapter(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	base := testutil.NewBase(t)
	ctx := t.Context()
	mongoDBContainer, err := mongoTestContainer.Run(ctx, testMongoDBImage)
	require.NoError(t, err)

	mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Disconnect(ctx))
	})

	database := client.Database("testdb")
	require.NoError(t, mongodb.EnsureSchema(ctx, database, false))

	adapter := mongodb.NewServerConnectionAdapter(database, base.Logger)

	now := time.Now()
	rec := func(server, ns string, uid uuid.UUID) *agentmodel.ServerConnection {
		return &agentmodel.ServerConnection{
			ServerID: server, UID: uid, InstanceUID: uuid.New(),
			Type: agentmodel.ConnectionTypeWebSocket, Namespace: ns,
			LastCommunicatedAt: now, SnapshotAt: now,
		}
	}

	a1, b1 := uuid.New(), uuid.New()

	t.Run("sync per server and list cluster-wide", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, adapter.SyncServerConnections(ctx, "server-a", now,
			[]*agentmodel.ServerConnection{rec("server-a", "default", a1)}, nil))
		require.NoError(t, adapter.SyncServerConnections(ctx, "server-b", now,
			[]*agentmodel.ServerConnection{rec("server-b", "default", b1)}, nil))

		resp, err := adapter.ListServerConnections(ctx, "default", "", time.Time{}, nil)
		require.NoError(t, err)
		assert.Len(t, resp.Items, 2)

		// Removing one server leaves the other untouched.
		require.NoError(t, adapter.RemoveServer(ctx, "server-a"))

		resp, err = adapter.ListServerConnections(ctx, "default", "", time.Time{}, nil)
		require.NoError(t, err)
		require.Len(t, resp.Items, 1)
		assert.Equal(t, "server-b", resp.Items[0].ServerID)
		assert.Equal(t, b1, resp.Items[0].UID)
	})

	t.Run("staleness cutoff excludes crashed servers", func(t *testing.T) {
		t.Parallel()
		// A stale heartbeat marks the server crashed, so its connections drop out.
		require.NoError(t, adapter.SyncServerConnections(ctx, "server-stale", now.Add(-10*time.Minute),
			[]*agentmodel.ServerConnection{rec("server-stale", "ns-stale", uuid.New())}, nil))

		resp, err := adapter.ListServerConnections(ctx, "ns-stale", "", now.Add(-90*time.Second), nil)
		require.NoError(t, err)
		assert.Empty(t, resp.Items)
	})

	t.Run("serverId filter restricts to one server", func(t *testing.T) {
		t.Parallel()

		x1, y1 := uuid.New(), uuid.New()
		require.NoError(t, adapter.SyncServerConnections(ctx, "server-x", now,
			[]*agentmodel.ServerConnection{rec("server-x", "ns-filter", x1)}, nil))
		require.NoError(t, adapter.SyncServerConnections(ctx, "server-y", now,
			[]*agentmodel.ServerConnection{rec("server-y", "ns-filter", y1)}, nil))

		resp, err := adapter.ListServerConnections(ctx, "ns-filter", "server-x", time.Time{}, nil)
		require.NoError(t, err)
		require.Len(t, resp.Items, 1)
		assert.Equal(t, "server-x", resp.Items[0].ServerID)
		assert.Equal(t, x1, resp.Items[0].UID)
	})
}
