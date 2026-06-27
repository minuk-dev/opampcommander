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
	adapter := mongodb.NewServerConnectionAdapter(database, base.Logger)

	now := time.Now()
	rec := func(server, ns string, uid uuid.UUID, snapshotAt time.Time) *agentmodel.ServerConnection {
		return &agentmodel.ServerConnection{
			ServerID: server, UID: uid, InstanceUID: uuid.New(),
			Type: agentmodel.ConnectionTypeWebSocket, Namespace: ns,
			LastCommunicatedAt: now, SnapshotAt: snapshotAt,
		}
	}

	a1, b1 := uuid.New(), uuid.New()

	t.Run("replace per server and list cluster-wide", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, adapter.ReplaceServerConnections(ctx, "server-a",
			[]*agentmodel.ServerConnection{rec("server-a", "default", a1, now)}))
		require.NoError(t, adapter.ReplaceServerConnections(ctx, "server-b",
			[]*agentmodel.ServerConnection{rec("server-b", "default", b1, now)}))

		resp, err := adapter.ListServerConnections(ctx, "default", time.Time{}, nil)
		require.NoError(t, err)
		assert.Len(t, resp.Items, 2)

		// Replacing one server's set leaves the other untouched.
		require.NoError(t, adapter.ReplaceServerConnections(ctx, "server-a", nil))

		resp, err = adapter.ListServerConnections(ctx, "default", time.Time{}, nil)
		require.NoError(t, err)
		require.Len(t, resp.Items, 1)
		assert.Equal(t, "server-b", resp.Items[0].ServerID)
		assert.Equal(t, b1, resp.Items[0].UID)
	})

	t.Run("staleness cutoff excludes old snapshots", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, adapter.ReplaceServerConnections(ctx, "server-stale",
			[]*agentmodel.ServerConnection{rec("server-stale", "ns-stale", uuid.New(), now.Add(-10*time.Minute))}))

		resp, err := adapter.ListServerConnections(ctx, "ns-stale", now.Add(-90*time.Second), nil)
		require.NoError(t, err)
		assert.Empty(t, resp.Items)
	})
}
