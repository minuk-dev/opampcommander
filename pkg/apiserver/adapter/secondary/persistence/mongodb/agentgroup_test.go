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
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
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

func TestAgentGroupMongoAdapter_Statistics_ConnectedIsStalenessAware(t *testing.T) {
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

	database := client.Database("testdb_agentgroup_stats")
	agentRepository := mongodb.NewAgentRepository(database, base.Logger)
	agentGroupAdapter := mongodb.NewAgentGroupRepository(database, base.Logger)

	// Fresh-connected -> counts as connected.
	fresh := agentmodel.NewAgent(uuid.New())
	fresh.UpdateLastCommunicationInfo(time.Now(), nil)
	require.NoError(t, agentRepository.PutAgent(ctx, fresh))

	// Explicitly disconnected -> not connected.
	require.NoError(t, agentRepository.PutAgent(ctx, agentmodel.NewAgent(uuid.New())))

	// Stale: flag still true but last report is past the staleness window. The
	// previous raw-flag aggregation would have miscounted this as connected; the
	// staleness-aware predicate must agree with the per-agent Connected field and
	// treat it as not connected.
	stale := agentmodel.NewAgent(uuid.New())
	stale.Status.Connected = true
	stale.Status.LastReportedAt = time.Now().Add(-1 * time.Hour)
	require.NoError(t, agentRepository.PutAgent(ctx, stale))

	// An empty selector matches every agent in the collection.
	group := agentmodel.NewAgentGroup("default", "all", agentmodel.OfAttributes(nil), time.Now(), "tester")
	_, err = agentGroupAdapter.PutAgentGroup(ctx, group.Metadata.Namespace, group.Metadata.Name, group)
	require.NoError(t, err)

	loaded, err := agentGroupAdapter.GetAgentGroup(ctx, group.Metadata.Namespace, group.Metadata.Name, nil)
	require.NoError(t, err)

	assert.Equal(t, 3, loaded.Status.NumAgents)
	assert.Equal(t, 1, loaded.Status.NumConnectedAgents)
	assert.Equal(t, 2, loaded.Status.NumNotConnectedAgents)
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
		agentGroup := agentmodel.NewAgentGroup(
			"default",
			"group-a",
			agentmodel.OfAttributes(map[string]string{"env": "prod", "team": "core"}),
			time.Now(),
			"tester",
		)

		putResult, err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name, agentGroup)
		require.NoError(t, err)

		// when
		loaded, err := adapter.GetAgentGroup(ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, putResult, loaded)
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
		got, err := adapter.GetAgentGroup(ctx, "default", "non-exist-group", nil)

		// then
		require.ErrorIs(t, err, port.ErrResourceNotExist)
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
		agentGroup := agentmodel.NewAgentGroup(
			"default",
			"group-single",
			agentmodel.OfAttributes(map[string]string{"env": "test"}),
			time.Now(),
			"tester",
		)
		putResult, err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name, agentGroup)
		require.NoError(t, err)

		// when
		resp, err := adapter.ListAgentGroups(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, putResult, resp.Items[0])
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
		agentGroups := make([]*agentmodel.AgentGroup, 3)

		for idx := range 3 {
			agentGroup := agentmodel.NewAgentGroup(
				"default",
				"group-"+uuid.NewString()[:8],
				agentmodel.OfAttributes(map[string]string{"idx": uuid.NewString()}),
				time.Now(),
				"tester",
			)
			agentGroups[idx] = agentGroup
			putResult, err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name, agentGroup)
			require.NoError(t, err)
			assert.Equal(t, agentGroup.Metadata.Name, putResult.Metadata.Name)
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
			assert.True(t, foundUIDs[agentGroup.Metadata.Name],
				"AgentGroup %s should be present in the list", agentGroup.Metadata.Name)
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
			agentGroup := agentmodel.NewAgentGroup(
				"default",
				"group-"+uuid.NewString()[:8],
				agentmodel.OfAttributes(map[string]string{"i": uuid.NewString()}),
				time.Now(),
				"tester",
			)
			_, err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name, agentGroup)
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
		agentGroup := agentmodel.NewAgentGroup(
			"default",
			"group-new",
			agentmodel.OfAttributes(map[string]string{"env": "staging", "version": "v1.0"}),
			time.Now(),
			"creator",
		)

		// when
		putResult, err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name, agentGroup)

		// then
		require.NoError(t, err)

		// Verify agent group was saved
		got, err := adapter.GetAgentGroup(ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name, nil)
		require.NoError(t, err)
		assert.Equal(t, putResult, got)
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
		originalGroup := agentmodel.NewAgentGroup(
			"default",
			"group-update",
			agentmodel.OfAttributes(map[string]string{"env": "dev"}),
			time.Now(),
			"creator",
		)
		_, err := adapter.PutAgentGroup(ctx, originalGroup.Metadata.Namespace, originalGroup.Metadata.Name, originalGroup)
		require.NoError(t, err)

		// when - update with new attributes
		updatedGroup := agentmodel.NewAgentGroup(
			"default",
			"group-update-new-name",
			agentmodel.OfAttributes(map[string]string{"env": "prod", "team": "backend"}),
			time.Now(),
			"updater",
		)
		_, err = adapter.PutAgentGroup(ctx, updatedGroup.Metadata.Namespace, updatedGroup.Metadata.Name, updatedGroup)
		require.NoError(t, err)

		// then
		got, err := adapter.GetAgentGroup(ctx, updatedGroup.Metadata.Namespace, updatedGroup.Metadata.Name, nil)
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
		agentGroup := agentmodel.NewAgentGroup(
			"default",
			"group-to-delete",
			agentmodel.OfAttributes(map[string]string{"env": "test"}),
			time.Now(),
			"creator",
		)
		_, err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name, agentGroup)
		require.NoError(t, err)

		// verify it's initially retrievable
		got, err := adapter.GetAgentGroup(ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name, nil)
		require.NoError(t, err)
		assert.False(t, got.IsDeleted())

		// when - soft delete
		deletedBy := "deleter"
		deletedAt := time.Now()
		agentGroup.MarkDeleted(deletedAt, deletedBy)
		savedGroup, err := adapter.PutAgentGroup(ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name, agentGroup)
		require.NoError(t, err)
		assert.True(t, savedGroup.IsDeleted())

		// then - should not be retrievable via normal get (soft deleted)
		_, err = adapter.GetAgentGroup(ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name, nil)
		require.ErrorIs(t, err, port.ErrResourceNotExist)

		// but should be retrievable with includeDeleted option
		deletedGroup, err := adapter.GetAgentGroup(
			ctx, agentGroup.Metadata.Namespace, agentGroup.Metadata.Name,
			&model.GetOptions{IncludeDeleted: true},
		)
		require.NoError(t, err)
		assert.True(t, deletedGroup.IsDeleted())
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

	originalGroup := agentmodel.NewAgentGroup(
		"default",
		"complex-group",
		agentmodel.OfAttributes(complexAttributes),
		time.Now(),
		"system-admin",
	)

	// when
	_, err := adapter.PutAgentGroup(ctx, originalGroup.Metadata.Namespace, originalGroup.Metadata.Name, originalGroup)
	require.NoError(t, err)

	loadedGroup, err := adapter.GetAgentGroup(ctx, originalGroup.Metadata.Namespace, originalGroup.Metadata.Name, nil)
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
