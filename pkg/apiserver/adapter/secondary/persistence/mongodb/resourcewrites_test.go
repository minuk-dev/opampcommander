//go:build integration

package mongodb_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestConditionalResourceWrites(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	ctx := t.Context()
	container, err := mongoTestContainer.Run(ctx, testMongoDBImage)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)
	uri, err := container.ConnectionString(ctx)
	require.NoError(t, err)
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Disconnect(ctx)) })

	db := client.Database("conditionalwrites")
	// Simulate the pre-upgrade namespace index; migration must replace it safely.
	_, err = db.Collection("namespaces").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "metadata.name", Value: 1}},
	})
	require.NoError(t, err)
	require.NoError(t, mongodb.EnsureSchema(ctx, db, false))
	logger := testutil.NewBase(t).Logger
	testutil.CheckResourceWrites(t, mongodb.NewAgentGroupRepository(db, logger),
		mongodb.NewAgentPackageRepository(db, logger), mongodb.NewAgentRemoteConfigRepository(db, logger),
		mongodb.NewNamespaceRepository(db, logger))
	// Migration upgrades legacy rows explicitly, remains idempotent, and never rewinds positive tokens.
	_, err = db.Collection("namespaces").InsertOne(ctx, bson.M{"metadata": bson.M{"name": "legacy"}})
	require.NoError(t, err)
	require.NoError(t, mongodb.EnsureSchema(ctx, db, false))
	repo := mongodb.NewNamespaceRepository(db, logger)
	legacy, err := repo.GetNamespace(ctx, "legacy", nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, legacy.Metadata.ResourceVersion)
	_, err = repo.PutNamespace(ctx, legacy)
	require.NoError(t, err)
	require.NoError(t, mongodb.EnsureSchema(ctx, db, false))
	legacy, err = repo.GetNamespace(ctx, "legacy", nil)
	require.NoError(t, err)
	require.EqualValues(t, 2, legacy.Metadata.ResourceVersion)
	// Dirty databases fail closed; schema migration never chooses a winner or deletes duplicates.
	dirty := client.Database("duplicates")
	_, err = dirty.Collection("namespaces").InsertMany(ctx, []any{
		bson.M{"metadata": bson.M{"name": "duplicate"}}, bson.M{"metadata": bson.M{"name": "duplicate"}},
	})
	require.NoError(t, err)
	require.Error(t, mongodb.EnsureSchema(ctx, dirty, false))
	count, err := dirty.Collection("namespaces").CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
}
