//go:build e2e

package apiserver_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// TestE2E_APIServer_DeleteAgent exercises the agent delete flow end-to-end through
// the same client library that opampctl uses against a live API server:
//   - a disconnected agent can be deleted (and is gone afterwards)
//   - a connected agent is rejected with 409 Conflict (and survives)
//   - deleting a non-existent agent returns 404 Not Found
func TestE2E_APIServer_DeleteAgent(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	dbName := "opampcommander_e2e_delete_agent_test"
	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, dbName)
	defer apiServer.Stop()

	apiServer.WaitForReady()

	opampClient := apiServer.Client()
	namespace := "default"

	// Given: One disconnected and one connected agent inserted directly into MongoDB.
	mongoClient, err := setupMongoDBClient(t, mongoServer.URI)
	require.NoError(t, err)

	defer func() { _ = mongoClient.Disconnect(ctx) }()

	disconnectedUID := uuid.New()
	connectedUID := uuid.New()

	collection := mongoClient.Database(dbName).Collection("agents")

	_, err = collection.InsertMany(ctx, []interface{}{
		bson.M{
			"metadata": bson.M{
				"instanceUid":       bson.Binary{Subtype: 0x04, Data: disconnectedUID[:]},
				"instanceUidString": disconnectedUID.String(),
				"namespace":         namespace,
			},
			"status": bson.M{
				"connected": false,
			},
			"spec": bson.M{},
		},
		bson.M{
			"metadata": bson.M{
				"instanceUid":       bson.Binary{Subtype: 0x04, Data: connectedUID[:]},
				"instanceUidString": connectedUID.String(),
				"namespace":         namespace,
			},
			"status": bson.M{
				"connected": true,
				// Recent heartbeat so the server considers it effectively connected.
				"lastCommunicatedAt": bson.NewDateTimeFromTime(time.Now()),
			},
			"spec": bson.M{},
		},
	})
	require.NoError(t, err)

	// Sanity check: both agents are visible through the API before deletion.
	_, err = opampClient.AgentService.GetAgent(ctx, namespace, disconnectedUID)
	require.NoError(t, err, "disconnected agent should exist before delete")
	_, err = opampClient.AgentService.GetAgent(ctx, namespace, connectedUID)
	require.NoError(t, err, "connected agent should exist before delete")

	t.Run("deletes a disconnected agent", func(t *testing.T) {
		// When: deleting the disconnected agent.
		err := opampClient.AgentService.DeleteAgent(ctx, namespace, disconnectedUID)

		// Then: it succeeds and the agent is gone.
		require.NoError(t, err)

		_, getErr := opampClient.AgentService.GetAgent(ctx, namespace, disconnectedUID)
		require.Error(t, getErr, "disconnected agent should be gone after delete")
		assert.Equal(t, http.StatusNotFound, responseStatusCode(t, getErr))
	})

	t.Run("rejects deleting a connected agent with 409", func(t *testing.T) {
		// When: deleting the connected agent.
		err := opampClient.AgentService.DeleteAgent(ctx, namespace, connectedUID)

		// Then: it is rejected with 409 Conflict and the agent survives.
		require.Error(t, err)
		assert.Equal(t, http.StatusConflict, responseStatusCode(t, err))

		_, getErr := opampClient.AgentService.GetAgent(ctx, namespace, connectedUID)
		require.NoError(t, getErr, "connected agent should still exist after rejected delete")
	})

	t.Run("returns 404 for a non-existent agent", func(t *testing.T) {
		// When: deleting an agent that does not exist.
		err := opampClient.AgentService.DeleteAgent(ctx, namespace, uuid.New())

		// Then: it returns 404 Not Found.
		require.Error(t, err)
		assert.Equal(t, http.StatusNotFound, responseStatusCode(t, err))
	})
}

// responseStatusCode extracts the HTTP status code from a client.ResponseError.
func responseStatusCode(t *testing.T, err error) int {
	t.Helper()

	var respErr *client.ResponseError

	require.ErrorAs(t, err, &respErr, "expected a client.ResponseError")

	return respErr.StatusCode
}
