package mongodb

import (
	"testing"

	"github.com/google/uuid"
	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// Use 4.4.10 because the test environment is raspberry pi (arm64)
	// but higher version does not support this hardware architecture.
	testMongoDBImage = "mongo:4.4.10"
)

func TestAgentMongoAdapter(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	ctx := t.Context()
	mongoDBContainer, err := mongoTestContainer.Run(
		ctx,
		testMongoDBImage,
	)
	require.NoError(t, err)

	mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoDBURI))
	require.NoError(t, err)
	t.Cleanup(func() {
		err := client.Disconnect(ctx)
		require.NoError(t, err)
	})

	database := client.Database("testdb")
	agentRepository := NewAgentRepository(database)

	t.Run("Happy case", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		instanceUID := uuid.New()
		agent := &domainmodel.Agent{
			InstanceUID: instanceUID,
		}
		err := agentRepository.PutAgent(ctx, agent)
		require.NoError(t, err)

		got, err := agentRepository.GetAgent(ctx, agent.InstanceUID)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, agent.InstanceUID, got.InstanceUID)
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		instanceUID := uuid.New()
		got, err := agentRepository.GetAgent(ctx, instanceUID)
		require.Error(t, err)
		require.Nil(t, got)
	})
}
