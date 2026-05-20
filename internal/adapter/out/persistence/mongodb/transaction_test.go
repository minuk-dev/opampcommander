package mongodb_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb"
)

var errSimulatedCascade = errors.New("simulated cascade failure")

// startReplicaSetMongo starts a mongo testcontainer configured as a single-node
// replica set, which is required to use MongoDB transactions.
//
// The testcontainers mongodb module initiates the replica set using the
// container's internal IP, which is not reachable from the host. We append
// ?directConnection=true so the driver bypasses SDAM topology discovery and
// talks to the published port directly; transactions still work because that
// single node is the replica-set primary.
func startReplicaSetMongo(t *testing.T) *mongo.Client {
	t.Helper()
	ctx := t.Context()

	container, err := mongoTestContainer.Run(
		ctx,
		testMongoDBImage,
		mongoTestContainer.WithReplicaSet("rs0"),
	)
	require.NoError(t, err)

	uri, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	if strings.Contains(uri, "?") {
		uri += "&directConnection=true"
	} else {
		uri += "?directConnection=true"
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Disconnect(ctx)
	})

	return client
}

func TestTransactionRunner_CommitsOnSuccess(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	client := startReplicaSetMongo(t)
	coll := client.Database("txdb_commit").Collection("docs")
	runner := mongodb.NewTransactionRunner(client)

	ctx := t.Context()

	err := runner.WithinTransaction(ctx, func(txCtx context.Context) error {
		_, insertErr := coll.InsertOne(txCtx, bson.M{"_id": "a", "v": 1})
		if insertErr != nil {
			return insertErr //nolint:wrapcheck // test
		}

		_, insertErr = coll.InsertOne(txCtx, bson.M{"_id": "b", "v": 2})
		if insertErr != nil {
			return insertErr //nolint:wrapcheck // test
		}

		return nil
	})
	require.NoError(t, err)

	cnt, err := coll.CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), cnt, "both inserts should be committed")
}

func TestTransactionRunner_RollsBackOnError(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	client := startReplicaSetMongo(t)
	coll := client.Database("txdb_rollback").Collection("docs")
	runner := mongodb.NewTransactionRunner(client)

	ctx := t.Context()

	err := runner.WithinTransaction(ctx, func(txCtx context.Context) error {
		_, insertErr := coll.InsertOne(txCtx, bson.M{"_id": "a", "v": 1})
		if insertErr != nil {
			return insertErr //nolint:wrapcheck // test
		}

		_, insertErr = coll.InsertOne(txCtx, bson.M{"_id": "b", "v": 2})
		if insertErr != nil {
			return insertErr //nolint:wrapcheck // test
		}

		return errSimulatedCascade
	})
	require.ErrorIs(t, err, errSimulatedCascade)

	cnt, err := coll.CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	assert.Zero(t, cnt, "all writes should be rolled back when callback errors")
}
