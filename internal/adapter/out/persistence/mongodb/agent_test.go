package mongodb

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/remoteconfig"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

const (
	// Use 4.4.10 because the test environment is raspberry pi (arm64)
	// but higher version does not support this hardware architecture.
	testMongoDBImage = "mongo:4.4.10"
)

func TestAgentMongoAdapter_GetAgent(t *testing.T) {
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
		// given
		agent := &model.Agent{
			InstanceUID:         instanceUID,
			Capabilities:        nil,
			Description:         nil,
			EffectiveConfig:     nil,
			PackageStatuses:     nil,
			ComponentHealth:     nil,
			RemoteConfig:        remoteconfig.New(),
			CustomCapabilities:  nil,
			AvailableComponents: nil,
			ReportFullState:     false,
		}
		err := agentRepository.PutAgent(ctx, agent)
		require.NoError(t, err)

		// when
		got, err := agentRepository.GetAgent(ctx, agent.InstanceUID)

		// then
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, agent.InstanceUID, got.InstanceUID)
		assert.Equal(t, agent.ReportFullState, got.ReportFullState)
	})

	t.Run("Agent not found", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		instanceUID := uuid.New()

		// when
		got, err := agentRepository.GetAgent(ctx, instanceUID)

		// then
		require.ErrorIs(t, err, domainport.ErrResourceNotExist)
		assert.Nil(t, got)
	})
}

func TestAgentMongoAdapter_ListAgents(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	t.Run("Empty list when no agents exist", func(t *testing.T) {
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

		database := client.Database("testdb_empty")
		agentRepository := NewAgentRepository(database)

		// when
		listResponse, err := agentRepository.ListAgents(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Equal(t, 0, len(listResponse.Items))
	})

	t.Run("Single agent in list", func(t *testing.T) {
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

		database := client.Database("testdb_single")
		agentRepository := NewAgentRepository(database)

		instanceUID := uuid.New()
		agent := &model.Agent{
			InstanceUID:         instanceUID,
			Capabilities:        nil,
			Description:         nil,
			EffectiveConfig:     nil,
			PackageStatuses:     nil,
			ComponentHealth:     nil,
			RemoteConfig:        remoteconfig.New(),
			CustomCapabilities:  nil,
			AvailableComponents: nil,
			ReportFullState:     false,
		}
		err = agentRepository.PutAgent(ctx, agent)
		require.NoError(t, err)

		// when
		listResponse, err := agentRepository.ListAgents(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Equal(t, 1, len(listResponse.Items))
		assert.Equal(t, instanceUID, listResponse.Items[0].InstanceUID)
	})

	t.Run("Multiple agents in list", func(t *testing.T) {
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

		database := client.Database("testdb_multiple")
		agentRepository := NewAgentRepository(database)

		// Create multiple agents
		agents := make([]*model.Agent, 3)
		for i := 0; i < 3; i++ {
			instanceUID := uuid.New()
			agent := &model.Agent{
				InstanceUID:         instanceUID,
				Capabilities:        nil,
				Description:         nil,
				EffectiveConfig:     nil,
				PackageStatuses:     nil,
				ComponentHealth:     nil,
				RemoteConfig:        remoteconfig.New(),
				CustomCapabilities:  nil,
				AvailableComponents: nil,
				ReportFullState:     false,
			}
			agents[i] = agent
			err = agentRepository.PutAgent(ctx, agent)
			require.NoError(t, err)
		}

		// when
		listResponse, err := agentRepository.ListAgents(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Equal(t, 3, len(listResponse.Items))

		// Check that all our agents are in the list
		foundUIDs := make(map[uuid.UUID]bool)
		for _, item := range listResponse.Items {
			foundUIDs[item.InstanceUID] = true
		}

		for _, agent := range agents {
			assert.True(t, foundUIDs[agent.InstanceUID], "Agent %s should be present in the list", agent.InstanceUID)
		}
	})

	t.Run("List with pagination options", func(t *testing.T) {
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

		database := client.Database("testdb_pagination")
		agentRepository := NewAgentRepository(database)

		// Create 5 agents
		for i := 0; i < 5; i++ {
			instanceUID := uuid.New()
			agent := &model.Agent{
				InstanceUID:         instanceUID,
				Capabilities:        nil,
				Description:         nil,
				EffectiveConfig:     nil,
				PackageStatuses:     nil,
				ComponentHealth:     nil,
				RemoteConfig:        remoteconfig.New(),
				CustomCapabilities:  nil,
				AvailableComponents: nil,
				ReportFullState:     false,
			}
			err = agentRepository.PutAgent(ctx, agent)
			require.NoError(t, err)
		}

		// when - list with limit of 3
		listOptions := &model.ListOptions{
			Limit: 3,
		}
		listResponse, err := agentRepository.ListAgents(ctx, listOptions)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.LessOrEqual(t, len(listResponse.Items), 3)

		// All returned agents should have valid UUIDs
		for _, item := range listResponse.Items {
			assert.NotEqual(t, uuid.Nil, item.InstanceUID)
		}
	})
}

func TestAgentMongoAdapter_PutAgent(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	t.Run("Happy case", func(t *testing.T) {
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

		instanceUID := uuid.New()
		agent := &model.Agent{
			InstanceUID:         instanceUID,
			Capabilities:        nil,
			Description:         nil,
			EffectiveConfig:     nil,
			PackageStatuses:     nil,
			ComponentHealth:     nil,
			RemoteConfig:        remoteconfig.New(),
			CustomCapabilities:  nil,
			AvailableComponents: nil,
			ReportFullState:     false,
		}

		// when
		err = agentRepository.PutAgent(ctx, agent)

		// then
		require.NoError(t, err)

		// Verify agent was saved
		got, err := agentRepository.GetAgent(ctx, instanceUID)
		require.NoError(t, err)
		assert.Equal(t, instanceUID, got.InstanceUID)
	})
}

func TestAgentMongoAdapter_ConfigShouldBeSameAfterSaveAndLoad(t *testing.T) {
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

	instanceUID := uuid.New()
	// when
	originalAgent := &model.Agent{
		InstanceUID:  instanceUID,
		Capabilities: nil,
		Description:  nil,
		EffectiveConfig: &model.AgentEffectiveConfig{
			ConfigMap: model.AgentConfigMap{
				ConfigMap: map[string]model.AgentConfigFile{
					"config.yaml": {
						Body:        []byte("key: value"),
						ContentType: "application/yaml",
					},
				},
			},
		},
		PackageStatuses:     nil,
		ComponentHealth:     nil,
		RemoteConfig:        remoteconfig.New(),
		CustomCapabilities:  nil,
		AvailableComponents: nil,
		ReportFullState:     false,
	}
	err = agentRepository.PutAgent(ctx, originalAgent)
	require.NoError(t, err)

	loadedAgent, err := agentRepository.GetAgent(ctx, instanceUID)
	require.NoError(t, err)

	// then
	assert.Equal(t, originalAgent.InstanceUID, loadedAgent.InstanceUID)
	assert.Equal(t, originalAgent.EffectiveConfig.ConfigMap.ConfigMap, loadedAgent.EffectiveConfig.ConfigMap.ConfigMap)
}
