package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestServerAdapter_GetServer(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	base := testutil.NewBase(t)
	ctx := context.Background()

	mongoDBContainer, err := mongoTestContainer.Run(ctx, testMongoDBImage)
	require.NoError(t, err)

	mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
	require.NoError(t, err)
	t.Cleanup(func() {
		err := client.Disconnect(ctx)
		require.NoError(t, err)
	})

	database := client.Database("testdb")
	adapter := mongodb.NewServerAdapter(base.Logger, database)

	t.Run("Get existing server", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		now := time.Now()
		server := &model.Server{
			ID:              "test-server-1",
			LastHeartbeatAt: now,
			Conditions:      []model.ServerCondition{},
		}
		server.MarkRegistered("test")

		err := adapter.PutServer(ctx, server)
		require.NoError(t, err)

		retrievedServer, err := adapter.GetServer(ctx, "test-server-1")
		require.NoError(t, err)
		assert.NotNil(t, retrievedServer)
		assert.Equal(t, "test-server-1", retrievedServer.ID)
	})

	t.Run("Get non-existing server", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		server, err := adapter.GetServer(ctx, "non-existing-server")
		require.ErrorIs(t, err, domainport.ErrResourceNotExist)
		assert.Nil(t, server)
	})
}

func TestServerAdapter_PutServer(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	base := testutil.NewBase(t)
	ctx := context.Background()

	mongoDBContainer, err := mongoTestContainer.Run(ctx, testMongoDBImage)
	require.NoError(t, err)

	mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
	require.NoError(t, err)
	t.Cleanup(func() {
		err := client.Disconnect(ctx)
		require.NoError(t, err)
	})

	database := client.Database("testdb")
	adapter := mongodb.NewServerAdapter(base.Logger, database)

	now := time.Now()
	server := &model.Server{
		ID:              "test-server",
		LastHeartbeatAt: now,
		Conditions:      []model.ServerCondition{},
	}
	server.MarkRegistered("test")

	err = adapter.PutServer(ctx, server)
	require.NoError(t, err)

	retrievedServer, err := adapter.GetServer(ctx, "test-server")
	require.NoError(t, err)
	assert.Equal(t, "test-server", retrievedServer.ID)
	assert.WithinDuration(t, now, retrievedServer.LastHeartbeatAt, time.Second)
	// Check that server has registered condition
	assert.True(t, retrievedServer.IsConditionTrue(model.ServerConditionTypeRegistered))
}

func TestServerAdapter_ListServers(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	base := testutil.NewBase(t)
	ctx := context.Background()

	mongoDBContainer, err := mongoTestContainer.Run(ctx, testMongoDBImage)
	require.NoError(t, err)

	mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
	require.NoError(t, err)
	t.Cleanup(func() {
		err := client.Disconnect(ctx)
		require.NoError(t, err)
	})

	database := client.Database("testdb")
	adapter := mongodb.NewServerAdapter(base.Logger, database)

	// Add multiple servers
	now := time.Now()
	for i := 1; i <= 3; i++ {
		server := &model.Server{
			ID:              "test-server-" + string(rune('0'+i)),
			LastHeartbeatAt: now,
			Conditions:      []model.ServerCondition{},
		}
		server.MarkRegistered("test")
		err := adapter.PutServer(ctx, server)
		require.NoError(t, err)
	}

	servers, err := adapter.ListServers(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(servers), 3)
}

func TestServer_IsAlive(t *testing.T) {
	t.Parallel()

	now := time.Now()
	timeout := 60 * time.Second

	tests := []struct {
		name        string
		server      *model.Server
		checkTime   time.Time
		expectedVal bool
	}{
		{
			name: "Server is alive - recent heartbeat",
			server: &model.Server{
				ID:              "server-1",
				LastHeartbeatAt: now.Add(-30 * time.Second),
				Conditions:      []model.ServerCondition{},
			},
			checkTime:   now,
			expectedVal: true,
		},
		{
			name: "Server is dead - old heartbeat",
			server: &model.Server{
				ID:              "server-2",
				LastHeartbeatAt: now.Add(-2 * time.Minute),
				Conditions:      []model.ServerCondition{},
			},
			checkTime:   now,
			expectedVal: false,
		},
		{
			name: "Server is barely alive - exactly at timeout",
			server: &model.Server{
				ID:              "server-3",
				LastHeartbeatAt: now.Add(-59 * time.Second),
				Conditions:      []model.ServerCondition{},
			},
			checkTime:   now,
			expectedVal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			isAlive := tt.server.IsAlive(tt.checkTime, timeout)
			assert.Equal(t, tt.expectedVal, isAlive)
		})
	}
}
