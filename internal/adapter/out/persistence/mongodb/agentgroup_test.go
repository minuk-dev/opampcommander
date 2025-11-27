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
		agentGroup := model.NewAgentGroup(
			"group-a",
			model.OfAttributes(map[string]string{"env": "prod", "team": "core"}),
			time.Now(),
			"tester",
		)

		err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Name, agentGroup)
		require.NoError(t, err)

		// when
		loaded, err := adapter.GetAgentGroup(ctx, agentGroup.Metadata.Name)

		// then
		require.NoError(t, err)
		assert.Equal(t, agentGroup.Metadata.Name, loaded.Metadata.Name)
		assert.Equal(t, agentGroup.Metadata.Attributes, loaded.Metadata.Attributes)
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
		agentGroup := model.NewAgentGroup(
			"group-single",
			model.OfAttributes(map[string]string{"env": "test"}),
			time.Now(),
			"tester",
		)
		err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Name, agentGroup)
		require.NoError(t, err)

		// when
		resp, err := adapter.ListAgentGroups(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, agentGroup.Metadata.Name, resp.Items[0].Metadata.Name)
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
		agentGroups := make([]*model.AgentGroup, 3)
		for idx := range 3 {
			agentGroup := model.NewAgentGroup(
				"group-"+uuid.NewString()[:8],
				model.OfAttributes(map[string]string{"idx": uuid.NewString()}),
				time.Now(),
				"tester",
			)
			agentGroups[idx] = agentGroup
			err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Name, agentGroup)
			require.NoError(t, err)
		}

		// when
		resp, err := adapter.ListAgentGroups(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Items, 3)

		// Check that all our agent groups are in the list
		foundUIDs := make(map[string]bool)
		for _, item := range resp.Items {
			foundUIDs[item.Metadata.Name] = true
		}

		for _, agentGroup := range agentGroups {
			assert.True(t, foundUIDs[agentGroup.Metadata.Name], "AgentGroup %s should be present in the list", agentGroup.Metadata.Name)
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
			agentGroup := model.NewAgentGroup(
				"group-"+uuid.NewString()[:8],
				model.OfAttributes(map[string]string{"i": uuid.NewString()}),
				time.Now(),
				"tester",
			)
			err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Name, agentGroup)
			require.NoError(t, err)
		}

		// when - list with limit of 2
		resp1, err := adapter.ListAgentGroups(ctx, &model.ListOptions{Limit: 2, Continue: ""})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(resp1.Items), 2)

		// when - list next page if continue token exists
		if resp1.Continue != "" {
			resp2, err := adapter.ListAgentGroups(ctx, &model.ListOptions{Limit: 2, Continue: resp1.Continue})
			require.NoError(t, err)
			assert.LessOrEqual(t, len(resp2.Items), 2)

			// Ensure no duplicate UIDs between pages
			page1UIDs := make(map[string]bool)
			for _, item := range resp1.Items {
				page1UIDs[item.Metadata.Name] = true
			}

			for _, item := range resp2.Items {
				assert.False(t, page1UIDs[item.Metadata.Name], "Name %s should not appear in both pages", item.Metadata.Name)
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
		agentGroup := model.NewAgentGroup(
			"group-new",
			model.OfAttributes(map[string]string{"env": "staging", "version": "v1.0"}),
			time.Now(),
			"creator",
		)

		// when
		err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Name, agentGroup)

		// then
		require.NoError(t, err)

		// Verify agent group was saved
		got, err := adapter.GetAgentGroup(ctx, agentGroup.Metadata.Name)
		require.NoError(t, err)
		assert.Equal(t, agentGroup.Metadata.Name, got.Metadata.Name)
		assert.Equal(t, agentGroup.Metadata.Attributes, got.Metadata.Attributes)
		assert.Equal(t, agentGroup.GetCreatedBy(), got.GetCreatedBy())
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
		originalGroup := model.NewAgentGroup(
			"group-update",
			model.OfAttributes(map[string]string{"env": "dev"}),
			time.Now(),
			"creator",
		)
		err := adapter.PutAgentGroup(ctx, originalGroup.Metadata.Name, originalGroup)
		require.NoError(t, err)

		// when - update with new attributes
		updatedGroup := model.NewAgentGroup(
			"group-update-new-name",
			model.OfAttributes(map[string]string{"env": "prod", "team": "backend"}),
			time.Now(),
			"updater",
		)
		err = adapter.PutAgentGroup(ctx, updatedGroup.Metadata.Name, updatedGroup)
		require.NoError(t, err)

		// then
		got, err := adapter.GetAgentGroup(ctx, updatedGroup.Metadata.Name)
		require.NoError(t, err)
		assert.Equal(t, updatedGroup.Metadata.Name, got.Metadata.Name)
		assert.Equal(t, updatedGroup.Metadata.Attributes, got.Metadata.Attributes)
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
		agentGroup := model.NewAgentGroup(
			"group-to-delete",
			model.OfAttributes(map[string]string{"env": "test"}),
			time.Now(),
			"creator",
		)
		err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Name, agentGroup)
		require.NoError(t, err)

		// when - soft delete
		deletedBy := "deleter"
		deletedAt := time.Now()
		agentGroup.MarkDeleted(deletedAt, deletedBy)
		err = adapter.PutAgentGroup(ctx, agentGroup.Metadata.Name, agentGroup)
		require.NoError(t, err)

		// then
		got, err := adapter.GetAgentGroup(ctx, agentGroup.Metadata.Name)
		require.NoError(t, err)
		assert.True(t, got.IsDeleted())
		assert.Equal(t, deletedBy, *got.GetDeletedBy())
		assert.NotNil(t, got.GetDeletedAt())
		// Check that deleted time is close to expected time (within 1 second)
		assert.WithinDuration(t, deletedAt, *got.GetDeletedAt(), time.Second)
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

	originalGroup := model.NewAgentGroup(
		"complex-group",
		model.OfAttributes(complexAttributes),
		time.Now(),
		"system-admin",
	)

	// when
	err := adapter.PutAgentGroup(ctx, originalGroup.Metadata.Name, originalGroup)
	require.NoError(t, err)

	loadedGroup, err := adapter.GetAgentGroup(ctx, originalGroup.Metadata.Name)
	require.NoError(t, err)

	// then
	assert.Equal(t, originalGroup.Metadata.Name, loadedGroup.Metadata.Name)
	assert.Equal(t, originalGroup.Metadata.Attributes, loadedGroup.Metadata.Attributes)
	assert.Equal(t, originalGroup.GetCreatedBy(), loadedGroup.GetCreatedBy())

	// Verify each attribute individually
	for key, expectedValue := range complexAttributes {
		actualValue, exists := loadedGroup.Metadata.Attributes[key]
		assert.True(t, exists, "Attribute key %s should exist", key)
		assert.Equal(t, expectedValue, actualValue, "Attribute value for key %s should match", key)
	}
}
