//go:build integration

package secondary_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx/fxtest"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter/secondary"
)

func TestNewMongoDatabaseWithoutDDLAuto(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	ctx := t.Context()
	container, err := mongoTestContainer.Run(ctx, "mongo:4.4.10")
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)
	uri, err := container.ConnectionString(ctx)
	require.NoError(t, err)
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Disconnect(ctx)) })

	settings := &config.ServerSettings{DatabaseSettings: config.DatabaseSettings{
		DatabaseName: "startup_validation", DDLAuto: false,
	}}
	db := client.Database(settings.DatabaseSettings.DatabaseName)
	_, err = db.Collection("namespaces").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "metadata.name", Value: 1}},
	})
	require.NoError(t, err)
	_, err = db.Collection("namespaces").InsertOne(ctx, bson.M{"metadata": bson.M{"name": "legacy"}})
	require.NoError(t, err)

	lifecycle := fxtest.NewLifecycle(t)
	_, err = secondary.NewMongoDatabase(client, settings, lifecycle)
	require.NoError(t, err)
	err = lifecycle.Start(ctx)
	require.ErrorContains(t, err, "resource schema is not ready")
	lifecycle.RequireStop()

	count, err := db.Collection("namespaces").CountDocuments(ctx, bson.M{
		"metadata.resourceVersion": bson.M{"$exists": false},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count, "DDL-disabled startup must not migrate data")

	// Provision the upgrade explicitly; the same DDL-disabled startup must now succeed.
	require.NoError(t, mongodb.EnsureSchema(ctx, db, false))
	lifecycle = fxtest.NewLifecycle(t)
	_, err = secondary.NewMongoDatabase(client, settings, lifecycle)
	require.NoError(t, err)
	lifecycle.RequireStart().RequireStop()
}
