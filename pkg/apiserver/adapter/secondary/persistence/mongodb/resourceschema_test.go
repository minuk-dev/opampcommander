//go:build integration

package mongodb_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
)

func TestValidateResourceSchema(t *testing.T) {
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

	for i, tt := range []struct {
		name    string
		options *options.IndexOptionsBuilder
	}{
		{name: "non-unique", options: options.Index()},
		{name: "sparse unique", options: options.Index().SetUnique(true).SetSparse(true)},
		{name: "partial unique", options: options.Index().SetUnique(true).SetPartialFilterExpression(
			bson.M{"metadata.deletedAt": bson.M{"$exists": true}})},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := client.Database(fmt.Sprintf("index_%d", i))
			require.NoError(t, mongodb.EnsureSchema(ctx, db, false))
			require.NoError(t, mongodb.ValidateResourceSchema(ctx, db))
			collection := db.Collection("namespaces")
			require.NoError(t, collection.Indexes().DropOne(ctx, "metadata.name_-1"))
			_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
				Keys: bson.D{{Key: "metadata.name", Value: -1}}, Options: tt.options.SetName("unsafe"),
			})
			require.NoError(t, err)

			err = mongodb.ValidateResourceSchema(ctx, db)
			require.ErrorContains(t, err, "namespaces is missing its full unique resource-key index")
		})
	}

	for i, tt := range []struct {
		name     string
		metadata bson.M
	}{
		{name: "missing", metadata: bson.M{"name": "legacy"}},
		{name: "zero", metadata: bson.M{"name": "legacy", "resourceVersion": int64(0)}},
		{name: "negative", metadata: bson.M{"name": "legacy", "resourceVersion": int64(-1)}},
		{name: "null", metadata: bson.M{"name": "legacy", "resourceVersion": nil}},
	} {
		t.Run(tt.name+" revision", func(t *testing.T) {
			t.Parallel()

			db := client.Database(fmt.Sprintf("revision_%d", i))
			require.NoError(t, mongodb.EnsureSchema(ctx, db, false))
			collection := db.Collection("namespaces")
			document := bson.M{"metadata": tt.metadata}
			_, err := collection.InsertOne(ctx, document)
			require.NoError(t, err)

			err = mongodb.ValidateResourceSchema(ctx, db)
			require.ErrorContains(t, err, "namespaces contains unmigrated resource versions")

			var stored struct {
				Metadata bson.M `bson:"metadata"`
			}

			err = collection.FindOne(ctx, bson.M{"metadata.name": "legacy"}).Decode(&stored)
			require.NoError(t, err)
			require.Equal(t, tt.metadata, stored.Metadata, "validation must leave legacy data unchanged")
		})
	}

	t.Run("ready schema with positive revisions", func(t *testing.T) {
		t.Parallel()

		db := client.Database("ready")
		require.NoError(t, mongodb.EnsureSchema(ctx, db, false))
		_, err := db.Collection("namespaces").InsertOne(ctx, bson.M{
			"metadata": bson.M{"name": "ready", "resourceVersion": int64(42)},
		})
		require.NoError(t, err)
		require.NoError(t, mongodb.ValidateResourceSchema(ctx, db))
	})
}
