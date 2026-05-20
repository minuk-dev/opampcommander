package mongodb_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errSimulatedCascade = errors.New("simulated cascade failure")

// startReplicaSetMongo starts a mongo testcontainer configured as a single-node
// replica set (required for MongoDB transactions) and returns a connected
// client.
func startReplicaSetMongo(t *testing.T) *mongo.Client {
	t.Helper()

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()

	ctx := t.Context()
	client, err := mongo.Connect(options.Client().ApplyURI(mongoServer.URI))
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

// TestTransactionRunner_ListInsideTransaction guards against issue:
// commonEntityAdapter.list/listWithFilter previously ran find + count in
// parallel goroutines sharing the same ctx; inside a transaction this would
// concurrently use a non-goroutine-safe *mongo.Session and corrupt session
// state or trip "transaction in progress" errors. This test exercises the
// real cascade pattern (put then list in the same transaction) to ensure
// runListQueries correctly serialises inside a session.
func TestTransactionRunner_ListInsideTransaction(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()
	ctx := t.Context()

	client, err := mongo.Connect(options.Client().ApplyURI(mongoServer.URI))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Disconnect(ctx)
	})

	database := client.Database("txdb_list")
	require.NoError(t, mongodb.EnsureSchema(ctx, database))

	repo := mongodb.NewAgentGroupRepository(database, base.Logger)
	runner := mongodb.NewTransactionRunner(client)

	group := agentmodel.NewAgentGroup(
		"team-a", "grp",
		agentmodel.OfAttributes(map[string]string{"env": "prod"}),
		time.Now(), "tester",
	)

	var listResp *model.ListResponse[*agentmodel.AgentGroup]

	err = runner.WithinTransaction(ctx, func(txCtx context.Context) error {
		_, putErr := repo.PutAgentGroup(txCtx, group.Metadata.Namespace, group.Metadata.Name, group)
		if putErr != nil {
			return putErr //nolint:wrapcheck // test
		}

		resp, listErr := repo.ListAgentGroups(txCtx, &model.ListOptions{})
		if listErr != nil {
			return listErr //nolint:wrapcheck // test
		}

		listResp = resp

		return nil
	})
	require.NoError(t, err, "put + list in the same transaction must not concurrently use the session")
	require.NotNil(t, listResp)
	assert.Len(t, listResp.Items, 1, "list inside the transaction must see the just-written row")
}
