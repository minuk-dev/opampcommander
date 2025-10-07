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

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func setupAgentGroupMongoAdapter(t *testing.T) (*mongo.Client, *mongodb.AgentGroupMongoAdapter) {
	t.Helper()
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

	database := client.Database("testdb_agentgroup")
	agentGroupAdapter := mongodb.NewAgentGroupRepository(database, base.Logger)

	return client, agentGroupAdapter
}

func TestAgentGroupMongoAdapter_GetAgentGroup(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	t.Run("Happy case", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		client, adapter := setupAgentGroupMongoAdapter(t)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		// given
		agentGroup := agentgroup.New(
			"group-a",
			agentgroup.OfAttributes(map[string]string{"env": "prod", "team": "core"}),
			time.Now(),
			"tester",
		)

		err := adapter.PutAgentGroup(ctx, agentGroup.Name, agentGroup)
		require.NoError(t, err)

		// when
		loaded, err := adapter.GetAgentGroup(ctx, agentGroup.Name)

		// then
		require.NoError(t, err)
		assert.Equal(t, agentGroup.UID, loaded.UID)
		assert.Equal(t, agentGroup.Name, loaded.Name)
		assert.Equal(t, agentGroup.Attributes, loaded.Attributes)
		assert.False(t, loaded.IsDeleted())
	})

	t.Run("Agent group not found", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		client, adapter := setupAgentGroupMongoAdapter(t)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		// when
		got, err := adapter.GetAgentGroup(ctx, "non-exist-group")

		// then
		require.ErrorIs(t, err, domainport.ErrResourceNotExist)
		assert.Nil(t, got)
	})
}

func TestAgentGroupMongoAdapter_ListAgentGroups(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	t.Run("Empty list when no agent groups exist", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		client, adapter := setupAgentGroupMongoAdapter(t)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		// when
		resp, err := adapter.ListAgentGroups(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.Items)
	})

	t.Run("Single agent group in list", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		client, adapter := setupAgentGroupMongoAdapter(t)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		// given
		agentGroup := agentgroup.New(
			"group-single",
			agentgroup.OfAttributes(map[string]string{"env": "test"}),
			time.Now(),
			"tester",
		)
		err := adapter.PutAgentGroup(ctx, agentGroup.Name, agentGroup)
		require.NoError(t, err)

		// when
		resp, err := adapter.ListAgentGroups(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, agentGroup.UID, resp.Items[0].UID)
		assert.Equal(t, agentGroup.Name, resp.Items[0].Name)
	})

	t.Run("Multiple agent groups in list", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		client, adapter := setupAgentGroupMongoAdapter(t)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		// given - create multiple groups
		agentGroups := make([]*agentgroup.AgentGroup, 3)
		for idx := range 3 {
			agentGroup := agentgroup.New(
				"group-"+uuid.NewString()[:8],
				agentgroup.OfAttributes(map[string]string{"idx": uuid.NewString()}),
				time.Now(),
				"tester",
			)
			agentGroups[idx] = agentGroup
			err := adapter.PutAgentGroup(ctx, agentGroup.Name, agentGroup)
			require.NoError(t, err)
		}

		// when
		resp, err := adapter.ListAgentGroups(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Items, 3)

		// Check that all our agent groups are in the list
		foundUIDs := make(map[uuid.UUID]bool)
		for _, item := range resp.Items {
			foundUIDs[item.UID] = true
		}

		for _, agentGroup := range agentGroups {
			assert.True(t, foundUIDs[agentGroup.UID], "AgentGroup %s should be present in the list", agentGroup.UID)
		}
	})

	t.Run("List with pagination options", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		client, adapter := setupAgentGroupMongoAdapter(t)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		// given - create 5 groups
		for range 5 {
			agentGroup := agentgroup.New(
				"group-"+uuid.NewString()[:8],
				agentgroup.OfAttributes(map[string]string{"i": uuid.NewString()}),
				time.Now(),
				"tester",
			)
			err := adapter.PutAgentGroup(ctx, agentGroup.Name, agentGroup)
			require.NoError(t, err)
		}

		// when - list with limit of 2
		resp1, err := adapter.ListAgentGroups(ctx, &model.ListOptions{Limit: 2, Continue: ""})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(resp1.Items), 2)

		// All returned agent groups should have valid UUIDs
		for _, item := range resp1.Items {
			assert.NotEqual(t, uuid.Nil, item.UID)
		}

		// when - list next page if continue token exists
		if resp1.Continue != "" {
			resp2, err := adapter.ListAgentGroups(ctx, &model.ListOptions{Limit: 2, Continue: resp1.Continue})
			require.NoError(t, err)
			assert.LessOrEqual(t, len(resp2.Items), 2)

			// Ensure no duplicate UIDs between pages
			page1UIDs := make(map[uuid.UUID]bool)
			for _, item := range resp1.Items {
				page1UIDs[item.UID] = true
			}

			for _, item := range resp2.Items {
				assert.False(t, page1UIDs[item.UID], "UID %s should not appear in both pages", item.UID)
			}
		}
	})
}

func TestAgentGroupMongoAdapter_PutAgentGroup(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	t.Run("Create new agent group", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		client, adapter := setupAgentGroupMongoAdapter(t)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		// given
		agentGroup := agentgroup.New(
			"group-new",
			agentgroup.OfAttributes(map[string]string{"env": "staging", "version": "v1.0"}),
			time.Now(),
			"creator",
		)

		// when
		err := adapter.PutAgentGroup(ctx, agentGroup.Name, agentGroup)

		// then
		require.NoError(t, err)

		// Verify agent group was saved
		got, err := adapter.GetAgentGroup(ctx, agentGroup.Name)
		require.NoError(t, err)
		assert.Equal(t, agentGroup.UID, got.UID)
		assert.Equal(t, agentGroup.Name, got.Name)
		assert.Equal(t, agentGroup.Attributes, got.Attributes)
		assert.Equal(t, agentGroup.CreatedBy, got.CreatedBy)
	})

	t.Run("Update existing agent group", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		client, adapter := setupAgentGroupMongoAdapter(t)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		// given - create initial agent group
		originalGroup := agentgroup.New(
			"group-update",
			agentgroup.OfAttributes(map[string]string{"env": "dev"}),
			time.Now(),
			"creator",
		)
		err := adapter.PutAgentGroup(ctx, originalGroup.Name, originalGroup)
		require.NoError(t, err)

		// when - update with new attributes
		updatedGroup := agentgroup.New(
			"group-update-new-name",
			agentgroup.OfAttributes(map[string]string{"env": "prod", "team": "backend"}),
			time.Now(),
			"updater",
		)
		updatedGroup.UID = originalGroup.UID // Keep same UID
		err = adapter.PutAgentGroup(ctx, updatedGroup.Name, updatedGroup)
		require.NoError(t, err)

		// then
		got, err := adapter.GetAgentGroup(ctx, updatedGroup.Name)
		require.NoError(t, err)
		assert.Equal(t, updatedGroup.UID, got.UID)
		assert.Equal(t, updatedGroup.Name, got.Name)
		assert.Equal(t, updatedGroup.Attributes, got.Attributes)
	})
}

func TestAgentGroupMongoAdapter_DeleteAgentGroup(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	t.Run("Soft delete agent group", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		client, adapter := setupAgentGroupMongoAdapter(t)
		t.Cleanup(func() {
			err := client.Disconnect(ctx)
			require.NoError(t, err)
		})

		// given
		agentGroup := agentgroup.New(
			"group-to-delete",
			agentgroup.OfAttributes(map[string]string{"env": "test"}),
			time.Now(),
			"creator",
		)
		err := adapter.PutAgentGroup(ctx, agentGroup.Name, agentGroup)
		require.NoError(t, err)

		// when - soft delete
		deletedBy := "deleter"
		deletedAt := time.Now()
		agentGroup.MarkDeleted(deletedAt, deletedBy)
		err = adapter.PutAgentGroup(ctx, agentGroup.Name, agentGroup)
		require.NoError(t, err)

		// then
		got, err := adapter.GetAgentGroup(ctx, agentGroup.Name)
		require.NoError(t, err)
		assert.True(t, got.IsDeleted())
		assert.Equal(t, deletedBy, *got.DeletedBy)
		assert.NotNil(t, got.DeletedAt)
		// Check that deleted time is close to expected time (within 1 second)
		assert.WithinDuration(t, deletedAt, *got.DeletedAt, time.Second)
	})
}

func TestAgentGroupMongoAdapter_AttributesShouldBeSameAfterSaveAndLoad(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	ctx := t.Context()
	client, adapter := setupAgentGroupMongoAdapter(t)
	t.Cleanup(func() {
		err := client.Disconnect(ctx)
		require.NoError(t, err)
	})

	// given - complex attributes
	complexAttributes := map[string]string{
		"environment":   "production",
		"datacenter":    "us-west-2",
		"team":          "platform",
		"cost-center":   "engineering-infra",
		"compliance":    "soc2",
		"monitoring":    "enabled",
		"backup":        "daily",
		"maintenance":   "weekends-only",
		"special-chars": "test-with-dashes_and_underscores.and.dots",
		"unicode":       "테스트-값",
	}

	originalGroup := agentgroup.New(
		"complex-group",
		agentgroup.OfAttributes(complexAttributes),
		time.Now(),
		"system-admin",
	)

	// when
	err := adapter.PutAgentGroup(ctx, originalGroup.Name, originalGroup)
	require.NoError(t, err)

	loadedGroup, err := adapter.GetAgentGroup(ctx, originalGroup.Name)
	require.NoError(t, err)

	// then
	assert.Equal(t, originalGroup.UID, loadedGroup.UID)
	assert.Equal(t, originalGroup.Name, loadedGroup.Name)
	assert.Equal(t, originalGroup.Attributes, loadedGroup.Attributes)
	assert.Equal(t, originalGroup.CreatedBy, loadedGroup.CreatedBy)

	// Verify each attribute individually
	for key, expectedValue := range complexAttributes {
		actualValue, exists := loadedGroup.Attributes[key]
		assert.True(t, exists, "Attribute key %s should exist", key)
		assert.Equal(t, expectedValue, actualValue, "Attribute value for key %s should match", key)
	}
}
