package mongodb_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	// Use 4.4.10 because the test environment is raspberry pi (arm64)
	// but higher version does not support this hardware architecture.
	testMongoDBImage = "mongo:4.4.10"
)

func TestAgentMongoAdapter_GetAgent(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	base := testutil.NewBase(t)
	ctx := t.Context()
	mongoDBContainer, err := mongoTestContainer.Run(
		ctx,
		testMongoDBImage,
	)
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
	agentRepository := mongodb.NewAgentRepository(database, base.Logger)

	t.Run("Happy case", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		instanceUID := uuid.New()
		// given
		agent := model.NewAgent(instanceUID)
		err := agentRepository.PutAgent(ctx, agent)
		require.NoError(t, err)

		// when
		got, err := agentRepository.GetAgent(ctx, agent.Metadata.InstanceUID)

		// then
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, agent.Metadata.InstanceUID, got.Metadata.InstanceUID)
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
	base := testutil.NewBase(t)

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

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_empty")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		// when
		listResponse, err := agentRepository.ListAgents(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Empty(t, listResponse.Items)
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

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_single")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		instanceUID := uuid.New()
		agent := model.NewAgent(instanceUID)
		err = agentRepository.PutAgent(ctx, agent)
		require.NoError(t, err)

		// when
		listResponse, err := agentRepository.ListAgents(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Len(t, listResponse.Items, 1)
		assert.Equal(t, instanceUID, listResponse.Items[0].Metadata.InstanceUID)
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

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_multiple")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		// Create multiple agents
		agents := make([]*model.Agent, 3)

		for idx := range 3 {
			instanceUID := uuid.New()
			agent := model.NewAgent(instanceUID)
			agents[idx] = agent
			err = agentRepository.PutAgent(ctx, agent)
			require.NoError(t, err)
		}

		// when
		listResponse, err := agentRepository.ListAgents(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Len(t, listResponse.Items, 3)

		// Check that all our agents are in the list
		foundUIDs := make(map[uuid.UUID]bool)
		for _, item := range listResponse.Items {
			foundUIDs[item.Metadata.InstanceUID] = true
		}

		for _, agent := range agents {
			assert.True(t, foundUIDs[agent.Metadata.InstanceUID])
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

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_pagination")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		// Create 5 agents
		for range 5 {
			instanceUID := uuid.New()
			agent := model.NewAgent(instanceUID)
			err = agentRepository.PutAgent(ctx, agent)
			require.NoError(t, err)
		}

		// when - list with limit of 3
		listOptions := &model.ListOptions{
			Limit:    3,
			Continue: "",
		}
		listResponse, err := agentRepository.ListAgents(ctx, listOptions)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.LessOrEqual(t, len(listResponse.Items), 3)

		// All returned agents should have valid UUIDs
		for _, item := range listResponse.Items {
			assert.NotEqual(t, uuid.Nil, item.Metadata.InstanceUID)
		}
	})
}

func TestAgentMongoAdapter_PutAgent(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	base := testutil.NewBase(t)

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

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		instanceUID := uuid.New()
		agent := model.NewAgent(instanceUID)

		// when
		err = agentRepository.PutAgent(ctx, agent)

		// then
		require.NoError(t, err)

		// Verify agent was saved
		got, err := agentRepository.GetAgent(ctx, instanceUID)
		require.NoError(t, err)
		assert.Equal(t, instanceUID, got.Metadata.InstanceUID)
	})
}

func TestAgentMongoAdapter_ConfigShouldBeSameAfterSaveAndLoad(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	base := testutil.NewBase(t)
	ctx := t.Context()
	mongoDBContainer, err := mongoTestContainer.Run(
		ctx,
		testMongoDBImage,
	)
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
	agentRepository := mongodb.NewAgentRepository(database, base.Logger)

	instanceUID := uuid.New()
	// when
	originalAgent := model.NewAgent(instanceUID)
	originalAgent.Status.EffectiveConfig = model.AgentEffectiveConfig{
		ConfigMap: model.AgentConfigMap{
			ConfigMap: map[string]model.AgentConfigFile{
				"config.yaml": {
					Body:        []byte("key: value"),
					ContentType: "application/yaml",
				},
			},
		},
	}
	err = agentRepository.PutAgent(ctx, originalAgent)
	require.NoError(t, err)

	loadedAgent, err := agentRepository.GetAgent(ctx, instanceUID)
	require.NoError(t, err)

	// then
	assert.Equal(t, originalAgent.Metadata.InstanceUID, loadedAgent.Metadata.InstanceUID)
	assert.Equal(t,
		originalAgent.Status.EffectiveConfig.ConfigMap.ConfigMap,
		loadedAgent.Status.EffectiveConfig.ConfigMap.ConfigMap,
	)
}

//nolint:maintidx // comprehensive integration test with multiple scenarios
func TestAgentMongoAdapter_ListAgentsBySelector(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	base := testutil.NewBase(t)

	t.Run("Empty list when no agents match selector", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		mongoDBContainer, err := mongoTestContainer.Run(
			ctx,
			testMongoDBImage,
		)
		require.NoError(t, err)

		mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
		require.NoError(t, err)

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_selector_empty")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		// Create an agent with different attributes
		instanceUID := uuid.New()
		agent := model.NewAgent(instanceUID, model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "other-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "windows",
			},
		}))
		err = agentRepository.PutAgent(ctx, agent)
		require.NoError(t, err)

		// when - search for non-existent selector
		selector := model.AgentSelector{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}
		listResponse, err := agentRepository.ListAgentsBySelector(ctx, selector, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Empty(t, listResponse.Items)
	})

	t.Run("Find agents with matching identifying attributes", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		mongoDBContainer, err := mongoTestContainer.Run(
			ctx,
			testMongoDBImage,
		)
		require.NoError(t, err)

		mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
		require.NoError(t, err)

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_selector_identifying")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		// Create agents with matching identifying attributes
		matchingAgent1 := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}))
		err = agentRepository.PutAgent(ctx, matchingAgent1)
		require.NoError(t, err)

		matchingAgent2 := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "darwin",
			},
		}))
		err = agentRepository.PutAgent(ctx, matchingAgent2)
		require.NoError(t, err)

		// Create agent with different identifying attributes
		nonMatchingAgent := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "other-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}))
		err = agentRepository.PutAgent(ctx, nonMatchingAgent)
		require.NoError(t, err)

		// when
		selector := model.AgentSelector{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
			NonIdentifyingAttributes: map[string]string{},
		}
		listResponse, err := agentRepository.ListAgentsBySelector(ctx, selector, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Len(t, listResponse.Items, 2)

		foundUIDs := make(map[uuid.UUID]bool)
		for _, item := range listResponse.Items {
			foundUIDs[item.Metadata.InstanceUID] = true
		}

		assert.True(t, foundUIDs[matchingAgent1.Metadata.InstanceUID])
		assert.True(t, foundUIDs[matchingAgent2.Metadata.InstanceUID])
		assert.False(t, foundUIDs[nonMatchingAgent.Metadata.InstanceUID])
	})

	t.Run("Find agents with matching non-identifying attributes", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		mongoDBContainer, err := mongoTestContainer.Run(
			ctx,
			testMongoDBImage,
		)
		require.NoError(t, err)

		mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
		require.NoError(t, err)

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_selector_non_identifying")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		// Create agents with matching non-identifying attributes
		matchingAgent1 := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "service-a",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}))
		err = agentRepository.PutAgent(ctx, matchingAgent1)
		require.NoError(t, err)

		matchingAgent2 := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "service-b",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}))
		err = agentRepository.PutAgent(ctx, matchingAgent2)
		require.NoError(t, err)

		// Create agent with different non-identifying attributes
		nonMatchingAgent := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "service-c",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "windows",
			},
		}))
		err = agentRepository.PutAgent(ctx, nonMatchingAgent)
		require.NoError(t, err)

		// when
		selector := model.AgentSelector{
			IdentifyingAttributes: map[string]string{},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}
		listResponse, err := agentRepository.ListAgentsBySelector(ctx, selector, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Len(t, listResponse.Items, 2)

		foundUIDs := make(map[uuid.UUID]bool)
		for _, item := range listResponse.Items {
			foundUIDs[item.Metadata.InstanceUID] = true
		}

		assert.True(t, foundUIDs[matchingAgent1.Metadata.InstanceUID])
		assert.True(t, foundUIDs[matchingAgent2.Metadata.InstanceUID])
		assert.False(t, foundUIDs[nonMatchingAgent.Metadata.InstanceUID])
	})

	t.Run("Find agents with matching both identifying and non-identifying attributes", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		mongoDBContainer, err := mongoTestContainer.Run(
			ctx,
			testMongoDBImage,
		)
		require.NoError(t, err)

		mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
		require.NoError(t, err)

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_selector_both")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		// Create agent that matches both attributes
		matchingAgent := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name":      "test-service",
				"service.namespace": "production",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type":   "linux",
				"host.name": "server-01",
			},
		}))
		err = agentRepository.PutAgent(ctx, matchingAgent)
		require.NoError(t, err)

		// Create agent that only matches identifying attributes
		partialMatch1 := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name":      "test-service",
				"service.namespace": "production",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type":   "windows",
				"host.name": "server-02",
			},
		}))
		err = agentRepository.PutAgent(ctx, partialMatch1)
		require.NoError(t, err)

		// Create agent that only matches non-identifying attributes
		partialMatch2 := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name":      "other-service",
				"service.namespace": "staging",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type":   "linux",
				"host.name": "server-01",
			},
		}))
		err = agentRepository.PutAgent(ctx, partialMatch2)
		require.NoError(t, err)

		// when
		selector := model.AgentSelector{
			IdentifyingAttributes: map[string]string{
				"service.name":      "test-service",
				"service.namespace": "production",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type":   "linux",
				"host.name": "server-01",
			},
		}
		listResponse, err := agentRepository.ListAgentsBySelector(ctx, selector, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Len(t, listResponse.Items, 1)
		assert.Equal(t, matchingAgent.Metadata.InstanceUID, listResponse.Items[0].Metadata.InstanceUID)
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

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_selector_pagination")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		// Create 5 agents with same selector
		for range 5 {
			agent := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
				IdentifyingAttributes: map[string]string{
					"service.name": "test-service",
				},
				NonIdentifyingAttributes: map[string]string{
					"os.type": "linux",
				},
			}))
			err = agentRepository.PutAgent(ctx, agent)
			require.NoError(t, err)
		}

		// when - list with limit of 3
		selector := model.AgentSelector{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}
		listOptions := &model.ListOptions{
			Limit:    3,
			Continue: "",
		}
		listResponse, err := agentRepository.ListAgentsBySelector(ctx, selector, listOptions)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.LessOrEqual(t, len(listResponse.Items), 3)
		assert.Equal(t, int64(2), listResponse.RemainingItemCount)

		// All returned agents should have valid UUIDs and match the selector
		for _, item := range listResponse.Items {
			assert.NotEqual(t, uuid.Nil, item.Metadata.InstanceUID)
			assert.Equal(t, "test-service", item.Metadata.Description.IdentifyingAttributes["service.name"])
			assert.Equal(t, "linux", item.Metadata.Description.NonIdentifyingAttributes["os.type"])
		}
	})

	t.Run("Invalid continue token", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		mongoDBContainer, err := mongoTestContainer.Run(
			ctx,
			testMongoDBImage,
		)
		require.NoError(t, err)

		mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
		require.NoError(t, err)

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_selector_invalid_token")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		// when
		selector := model.AgentSelector{
			IdentifyingAttributes:    map[string]string{},
			NonIdentifyingAttributes: map[string]string{},
		}
		listOptions := &model.ListOptions{
			Limit:    10,
			Continue: "invalid-token",
		}
		listResponse, err := agentRepository.ListAgentsBySelector(ctx, selector, listOptions)

		// then
		require.Error(t, err)
		assert.Nil(t, listResponse)
		assert.Contains(t, err.Error(), "invalid continue token")
	})
}

func TestAgentMongoAdapter_NewInstanceUID(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	
	t.Run("Agent with NewInstanceUID", func(t *testing.T) {
		t.Parallel()
		base := testutil.NewBase(t)
		ctx := t.Context()
		mongoDBContainer, err := mongoTestContainer.Run(
			ctx,
			testMongoDBImage,
		)
		require.NoError(t, err)

		mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
		require.NoError(t, err)

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_newinstanceuid")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		// given
		instanceUID := uuid.New()
		newInstanceUID := []byte("new-instance-uid-123")
		agent := model.NewAgent(instanceUID)
		agent.Spec.NewInstanceUID = newInstanceUID

		// when - Save agent
		err = agentRepository.PutAgent(ctx, agent)
		require.NoError(t, err)

		// then - Retrieve and verify
		retrievedAgent, err := agentRepository.GetAgent(ctx, instanceUID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedAgent)
		assert.Equal(t, newInstanceUID, retrievedAgent.Spec.NewInstanceUID)
		assert.True(t, retrievedAgent.HasNewInstanceUID())
		assert.Equal(t, newInstanceUID, retrievedAgent.NewInstanceUID())
	})

	t.Run("Agent without NewInstanceUID", func(t *testing.T) {
		t.Parallel()
		base := testutil.NewBase(t)
		ctx := t.Context()
		mongoDBContainer, err := mongoTestContainer.Run(
			ctx,
			testMongoDBImage,
		)
		require.NoError(t, err)

		mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
		require.NoError(t, err)

		client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
		require.NoError(t, err)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		database := client.Database("testdb_nonewinstanceuid")
		agentRepository := mongodb.NewAgentRepository(database, base.Logger)

		// given
		instanceUID := uuid.New()
		agent := model.NewAgent(instanceUID)
		// NewInstanceUID is not set (default nil/empty)

		// when - Save agent
		err = agentRepository.PutAgent(ctx, agent)
		require.NoError(t, err)

		// then - Retrieve and verify
		retrievedAgent, err := agentRepository.GetAgent(ctx, instanceUID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedAgent)
		assert.Nil(t, retrievedAgent.Spec.NewInstanceUID)
		assert.False(t, retrievedAgent.HasNewInstanceUID())
		assert.Nil(t, retrievedAgent.NewInstanceUID())
	})
}
