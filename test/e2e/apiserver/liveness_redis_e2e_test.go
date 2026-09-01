//go:build e2e

package apiserver_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// livenessFlushInterval is dialled well below the production default so the test can
// watch the database converge without waiting a minute for each cycle.
const livenessFlushInterval = 3 * time.Second

// TestE2E_APIServer_RedisLivenessSurvivesRedisOutage is the degradation guarantee of
// the liveness fast tier, exercised end to end: with Redis as the shared tier,
// killing Redis mid-flight must not fail a single request, must not flip a live
// agent offline, and must leave MongoDB converging while Redis stays down.
//
// The failure this guards against is specific. If liveness lived only in Redis and
// Redis died, every agent's stored LastReportedAt would be as old as the last
// write-behind flush; past the 90s staleness window the API would report the entire
// fleet as disconnected while its WebSockets were perfectly fine.
func TestE2E_APIServer_RedisLivenessSurvivesRedisOutage(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	dbName := "opampcommander_e2e_liveness_redis"
	mongoServer := base.StartMongoDB()
	redisServer := base.StartRedis()

	apiServer := base.StartAPIServerWithRedisLiveness(
		mongoServer.URI, redisServer.Endpoint, dbName, livenessFlushInterval)
	defer apiServer.Stop()

	apiServer.WaitForReady()

	otelCollector := base.StartOTelCollector(apiServer.Port)
	defer func() { _ = otelCollector.Terminate(ctx) }()

	opampClient := apiServer.Client()

	// Given: the collector is registered and reporting through the Redis-backed tier.
	require.Eventually(t, func() bool {
		agent, err := tryGetAgentByIDWithClient(opampClient, otelCollector.UID)

		return err == nil && agent != nil && agent.Status.Connected
	}, 3*time.Minute, time.Second, "the collector should register and report as connected")

	mongoClient, err := setupMongoDBClient(t, mongoServer.URI)
	require.NoError(t, err)

	defer func() { _ = mongoClient.Disconnect(ctx) }()

	// And: its liveness has reached MongoDB, so the outage starts from a converged
	// state rather than an empty one.
	require.Eventually(t, func() bool {
		return !storedLastReportedAt(t, mongoClient, dbName, otelCollector.UID).IsZero()
	}, time.Minute, time.Second, "the write-behind flush should reach mongodb before the outage")

	beforeOutage := storedLastReportedAt(t, mongoClient, dbName, otelCollector.UID)

	// When: Redis is killed while the collector keeps heartbeating.
	require.NoError(t, redisServer.Terminate(ctx))

	// Then: no request fails, and the agent never flips offline. The staleness window
	// is 90s, so a fleet-wide flip would surface well inside this loop.
	const outageWatch = 2 * time.Minute

	deadline := time.Now().Add(outageWatch)
	for time.Now().Before(deadline) {
		agent, err := tryGetAgentByIDWithClient(opampClient, otelCollector.UID)
		require.NoError(t, err, "a Redis outage must never surface as a failed API request")
		require.NotNil(t, agent, "the agent must stay visible while Redis is down")
		require.True(t, agent.Status.Connected,
			"a live agent must not read as disconnected just because the fast tier is gone")

		time.Sleep(5 * time.Second)
	}

	// And: MongoDB kept converging during the outage. The node-local fallback took
	// over the write path, so the stored liveness advanced instead of freezing at the
	// last flush before Redis died.
	afterOutage := storedLastReportedAt(t, mongoClient, dbName, otelCollector.UID)
	assert.True(t, afterOutage.After(beforeOutage),
		"mongodb should keep advancing during the outage; before=%s after=%s", beforeOutage, afterOutage)

	// And: the server reports the degradation without failing a probe — an optional
	// accelerator must never be the reason a process is restarted.
	assert.True(t, apiServer.IsReady(), "readiness must survive the loss of an optional dependency")
}

// storedLastReportedAt reads the agent's LastReportedAt straight from MongoDB.
//
// It deliberately bypasses the API: the assertion is about what is durably stored,
// not what the read path merges in from the fast tier.
func storedLastReportedAt(
	t *testing.T,
	client *mongo.Client,
	dbName string,
	agentUID uuid.UUID,
) time.Time {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	uuidBytes, err := agentUID.MarshalBinary()
	require.NoError(t, err)

	// The stored field is lastCommunicatedAt; Status.LastReportedAt is its domain name.
	var result struct {
		Status struct {
			LastCommunicatedAt time.Time `bson:"lastCommunicatedAt"`
		} `bson:"status"`
	}

	err = client.Database(dbName).Collection("agents").
		FindOne(ctx, bson.M{"metadata.instanceUid": bson.Binary{
			Subtype: 0x04,
			Data:    uuidBytes,
		}}).Decode(&result)
	if err != nil {
		return time.Time{}
	}

	return result.Status.LastCommunicatedAt
}
