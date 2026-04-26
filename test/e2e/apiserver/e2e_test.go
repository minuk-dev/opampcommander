//go:build e2e

package apiserver_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	testMongoDBImage = "mongo:4.4.10"
)

func TestE2E_APIServer_WithOTelCollector(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()
	dbName := "opampcommander_e2e_test_single"

	apiServer := base.StartAPIServer(mongoURI, dbName)
	defer apiServer.Stop()

	apiServer.WaitForReady()

	// Given: OTel Collector is started
	otelCollector := base.StartOTelCollector(apiServer.Port)
	defer func() { _ = otelCollector.Terminate(ctx) }()

	opampClient := apiServer.Client()
	namespace := "default"

	// When: Collector reports via OpAMP
	assert.Eventually(t, func() bool {
		agentsResp, err := opampClient.AgentService.ListAgents(ctx, namespace)
		if err != nil {
			return false
		}

		return len(agentsResp.Items) > 0
	}, 3*time.Minute, 1*time.Second, "At least one agent should register within timeout")

	assert.Eventually(t, func() bool {
		// Then: Collector has complete metadata
		agentsResp, err := opampClient.AgentService.ListAgents(ctx, namespace)
		if err != nil {
			return false
		}

		agentList := lo.Filter(agentsResp.Items, func(a v1.Agent, _ int) bool {
			return a.Metadata.InstanceUID == otelCollector.UID
		})

		if len(agentList) == 0 {
			return false
		}

		agent := agentList[0]

		hasDescription := len(agent.Metadata.Description.IdentifyingAttributes) > 0 ||
			len(agent.Metadata.Description.NonIdentifyingAttributes) > 0
		if !hasDescription {
			return false // Agent does not have description yet
		}

		return agent.Metadata.Capabilities != 0 // Agent should have capabilities
	}, 30*time.Second, 1*time.Second, "Agent metadata should be complete within timeout")

	// Then: Agent is retrievable by ID
	assert.Eventually(t, func() bool {
		agent, err := opampClient.AgentService.GetAgent(ctx, namespace, otelCollector.UID)
		if err != nil {
			return false
		}

		return agent.Metadata.InstanceUID == otelCollector.UID
	}, 30*time.Second, 1*time.Second, "Agent should be retrievable by ID within timeout")
}

func TestE2E_APIServer_MultipleCollectors(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up
	mongoContainer, mongoURI := startMongoDB(t)

	defer func() { _ = mongoContainer.Terminate(t.Context()) }()

	apiPort := base.GetFreeTCPPort()

	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_test_multi")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: Multiple collectors are started
	numCollectors := 3
	collectors := make([]*testutil.OTelCollector, numCollectors)

	for i := range numCollectors {
		collectors[i] = base.StartOTelCollector(apiPort)
	}

	defer func() {
		for _, c := range collectors {
			_ = c.Terminate(t.Context())
		}
	}()

	// When: All collectors report via OpAMP
	assert.Eventually(t, func() bool {
		agents := listAgents(t, apiBaseURL)

		if len(agents) < numCollectors {
			return false
		}

		foundCount := 0

		for _, c := range collectors {
			if findAgentByUID(agents, c.UID) != nil {
				foundCount++
			}
		}

		return foundCount == numCollectors
	}, 30*time.Second, 1*time.Second, "All collectors should register within timeout")
}

func startMongoDB(t *testing.T) (testcontainers.Container, string) {
	t.Helper()

	container, err := mongoTestContainer.Run(t.Context(), testMongoDBImage)
	require.NoError(t, err)

	uri, err := container.ConnectionString(t.Context())
	require.NoError(t, err)

	return container, uri
}

func TestE2E_APIServer_SequenceNum(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	mongoContainer, mongoURI := startMongoDB(t)

	defer func() { _ = mongoContainer.Terminate(ctx) }()

	dbName := "opampcommander_e2e_sequencenum_test"

	apiServer := base.StartAPIServer(mongoURI, dbName)
	defer apiServer.Stop()

	apiServer.WaitForReady()

	// Setup MongoDB client to verify data directly
	mongoClient, err := setupMongoDBClient(t, mongoURI)
	require.NoError(t, err)

	defer func() { _ = mongoClient.Disconnect(ctx) }()

	// Given: OTel Collector is started
	collector := base.StartOTelCollector(apiServer.Port)
	defer func() { _ = collector.Terminate(ctx) }()

	// When: Collector reports via OpAMP multiple times
	t.Log("Waiting for collector to register...")
	require.Eventually(t, func() bool {
		agents := listAgents(t, apiServer.Endpoint)

		return len(agents) > 0
	}, 2*time.Minute, 1*time.Second, "Agent should register within timeout")

	// Then: SequenceNum should be visible through API and incrementing
	t.Log("Verifying SequenceNum through API...")

	var previousSeqNum uint64

	require.Eventually(t, func() bool {
		agent := getAgentByID(t, apiServer.Endpoint, collector.UID)

		// First check: SequenceNum should be present and non-zero
		if agent.Status.SequenceNum == 0 {
			t.Logf("SequenceNum is still 0, waiting...")

			return false
		}

		// Track sequence number progression
		if previousSeqNum == 0 {
			previousSeqNum = agent.Status.SequenceNum
			t.Logf("Initial SequenceNum: %d", agent.Status.SequenceNum)

			return false // Need to wait for next report to confirm increment
		}

		// Verify it's incrementing
		if agent.Status.SequenceNum > previousSeqNum {
			t.Logf("SequenceNum incremented from %d to %d", previousSeqNum, agent.Status.SequenceNum)

			return true
		}

		t.Logf("SequenceNum: %d (previous: %d)", agent.Status.SequenceNum, previousSeqNum)
		previousSeqNum = agent.Status.SequenceNum

		return false
	}, 90*time.Second, 3*time.Second, "SequenceNum should increment through API")

	// Then: Verify SequenceNum in MongoDB directly
	t.Log("Verifying SequenceNum in MongoDB...")
	verifySequenceNumInMongoDB(t, mongoClient, dbName, collector.UID)

	// Then: Final verification - get multiple reports and ensure monotonic increase
	t.Log("Final verification of SequenceNum progression...")

	seqNums := make([]uint64, 0, 5)

	for i := range 5 {
		time.Sleep(3 * time.Second)

		agent := getAgentByID(t, apiServer.Endpoint, collector.UID)
		seqNums = append(seqNums, agent.Status.SequenceNum)
		t.Logf("Sample %d: SequenceNum = %d", i+1, agent.Status.SequenceNum)
	}

	// Verify monotonic increase
	for i := 1; i < len(seqNums); i++ {
		assert.GreaterOrEqual(t, seqNums[i], seqNums[i-1],
			"SequenceNum should be monotonically increasing or equal")
	}

	// At least some should have increased
	assert.GreaterOrEqual(t, seqNums[len(seqNums)-1], seqNums[0],
		"SequenceNum should have increased over time")
}

func setupMongoDBClient(t *testing.T, mongoURI string) (*mongo.Client, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client, nil
}

func verifySequenceNumInMongoDB(t *testing.T, client *mongo.Client, dbName string, agentUID uuid.UUID) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	collection := client.Database(dbName).Collection("agents")

	// Query for the agent - InstanceUID is stored as binary UUID
	var result struct {
		Status struct {
			SequenceNum uint64 `bson:"sequenceNum"`
		} `bson:"status"`
	}

	// Convert UUID to BSON Binary format
	uuidBytes, err := agentUID.MarshalBinary()
	require.NoError(t, err)

	filter := bson.M{"metadata.instanceUid": bson.Binary{
		Subtype: 0x04, // UUID subtype
		Data:    uuidBytes,
	}}

	err = collection.FindOne(ctx, filter).Decode(&result)
	require.NoError(t, err, "Should find agent in MongoDB")

	assert.Positive(t, result.Status.SequenceNum,
		"SequenceNum in MongoDB should be greater than 0")

	t.Logf("SequenceNum in MongoDB: %d", result.Status.SequenceNum)
}

func TestE2E_APIServer_SearchAgents(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	mongoContainer, mongoURI := startMongoDB(t)

	defer func() {
		_ = mongoContainer.Terminate(ctx)
	}()

	apiPort := base.GetFreeTCPPort()

	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_search_test")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	opampClient := createOpampClient(t, apiBaseURL)

	// Create multiple agents with known UIDs
	agent1UID := uuid.MustParse("12345678-1234-1234-1234-123456789012")
	agent2UID := uuid.MustParse("12345678-5678-5678-5678-567856785678")
	agent3UID := uuid.MustParse("abcdef01-2345-6789-abcd-ef0123456789")

	// Insert agents via MongoDB directly to simulate existing agents
	mongoClient, err := setupMongoDBClient(t, mongoURI)
	require.NoError(t, err)

	defer func() {
		_ = mongoClient.Disconnect(ctx)
	}()

	collection := mongoClient.Database("opampcommander_search_test").Collection("agents")

	// Insert test agents
	agents := []interface{}{
		bson.M{
			"metadata": bson.M{
				"instanceUid":       bson.Binary{Subtype: 0x04, Data: agent1UID[:]},
				"instanceUidString": agent1UID.String(),
				"namespace":         "default",
			},
			"status": bson.M{
				"connected": true,
			},
			"spec": bson.M{},
		},
		bson.M{
			"metadata": bson.M{
				"instanceUid":       bson.Binary{Subtype: 0x04, Data: agent2UID[:]},
				"instanceUidString": agent2UID.String(),
				"namespace":         "default",
			},
			"status": bson.M{
				"connected": true,
			},
			"spec": bson.M{},
		},
		bson.M{
			"metadata": bson.M{
				"instanceUid":       bson.Binary{Subtype: 0x04, Data: agent3UID[:]},
				"instanceUidString": agent3UID.String(),
				"namespace":         "default",
			},
			"status": bson.M{
				"connected": false,
			},
			"spec": bson.M{},
		},
	}

	_, err = collection.InsertMany(ctx, agents)
	require.NoError(t, err)

	// When: Search by prefix "12345678-1234"
	t.Log("Searching for agents with prefix '12345678-1234'...")
	searchResp := searchAgents(t, opampClient, "12345678-1234")

	// Then: Should find only agent1
	assert.Len(t, searchResp.Items, 1)
	assert.Equal(t, agent1UID, searchResp.Items[0].Metadata.InstanceUID)

	// When: Search by prefix "1234"
	t.Log("Searching for agents with prefix '1234'...")
	searchResp = searchAgents(t, opampClient, "1234")

	// Then: Should find agent1 and agent2
	assert.Len(t, searchResp.Items, 2)

	// When: Search by prefix "abcd" (case-insensitive)
	t.Log("Searching for agents with prefix 'ABCD' (uppercase)...")
	searchResp = searchAgents(t, opampClient, "ABCD")

	// Then: Should find agent3
	assert.Len(t, searchResp.Items, 1)
	assert.Equal(t, agent3UID, searchResp.Items[0].Metadata.InstanceUID)

	// When: Search with no matches
	t.Log("Searching for agents with non-existent prefix...")
	searchResp = searchAgents(t, opampClient, "zzzzz")

	// Then: Should return empty result
	assert.Empty(t, searchResp.Items)

	// When: Search with pagination
	t.Log("Searching with pagination (limit=1)...")
	searchResp = searchAgentsWithLimit(t, opampClient, "1234", 1)

	// Then: Should return 1 result and have continue token
	assert.Len(t, searchResp.Items, 1)
	assert.NotEmpty(t, searchResp.Metadata.Continue)
}

func searchAgents(t *testing.T, c *client.Client, query string) *v1.ListResponse[v1.Agent] {
	t.Helper()

	return searchAgentsWithLimit(t, c, query, 0)
}

func searchAgentsWithLimit(t *testing.T, c *client.Client, query string, limit int) *v1.ListResponse[v1.Agent] {
	t.Helper()

	var opts []client.ListOption
	if limit > 0 {
		opts = append(opts, client.WithLimit(limit))
	}

	result, err := c.AgentService.SearchAgents(t.Context(), "default", query, opts...)
	require.NoError(t, err)

	return result
}

// TestE2E_ConnectionType_HTTPAndWebSocket tests that HTTP and WebSocket connections
// are correctly identified and reported via the connections API.
func TestE2E_ConnectionType_HTTPAndWebSocket(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	mongoContainer, mongoURI := startMongoDB(t)

	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()

	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_conntype_test")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	opampClient := createOpampClient(t, apiBaseURL)

	// Given: Two OTel Collectors - one with HTTP polling, one with WebSocket
	httpCollector := base.StartOTelCollectorHTTP(apiPort)
	defer func() { _ = httpCollector.Terminate(ctx) }()

	wsCollector := base.StartOTelCollector(apiPort)
	defer func() { _ = wsCollector.Terminate(ctx) }()

	// When: Both collectors connect
	assert.Eventually(t, func() bool {
		agents := listAgents(t, apiBaseURL)

		return len(agents) >= 2
	}, 2*time.Minute, 1*time.Second, "Both collectors should register")

	// Then: Verify agents are registered
	agents := listAgents(t, apiBaseURL)
	require.GreaterOrEqual(t, len(agents), 2, "Both collectors should register as agents")

	// Find our collectors in the agents list
	httpAgent := findAgentByUID(agents, httpCollector.UID)
	wsAgent := findAgentByUID(agents, wsCollector.UID)

	require.NotNil(t, httpAgent, "HTTP collector should be registered as agent")
	require.NotNil(t, wsAgent, "WebSocket collector should be registered as agent")

	// Then: Check that WebSocket connection is active
	// Note: HTTP polling connections are not persistent, so they won't appear in connections list
	assert.Eventually(t, func() bool {
		connections := listConnections(t, opampClient)
		t.Logf("Found %d active connections", len(connections))

		for _, conn := range connections {
			t.Logf("Connection InstanceUID: %s, Type: %s", conn.InstanceUID, conn.Type)

			if conn.InstanceUID == wsCollector.UID && conn.Type == "WebSocket" {
				t.Logf("WebSocket collector connection found")

				return true
			}
		}

		return false
	}, 30*time.Second, 1*time.Second, "WebSocket connection should be active")
}

// listConnections retrieves all connections from the API.
func listConnections(t *testing.T, c *client.Client) []v1.Connection {
	t.Helper()

	resp, err := c.ConnectionService.ListConnections(t.Context(), "default")
	require.NoError(t, err)

	return resp.Items
}


// TestE2E_AgentPackage_CRUD tests the CRUD operations for agent packages via the API.
func TestE2E_AgentPackage_CRUD(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	mongoContainer, mongoURI := startMongoDB(t)

	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()

	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_agentpackage_test")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	opampClient := createOpampClient(t, apiBaseURL)

	// Test Create Agent Package
	t.Run("Create AgentPackage", func(t *testing.T) {
		pkg := createAgentPackage(t, opampClient, "test-package-1", "TopLevelPackageName", "1.0.0")
		assert.Equal(t, "test-package-1", pkg.Metadata.Name)
		assert.Equal(t, "TopLevelPackageName", pkg.Spec.PackageType)
		assert.Equal(t, "1.0.0", pkg.Spec.Version)
	})

	// Test List Agent Packages
	t.Run("List AgentPackages", func(t *testing.T) {
		// Create another package
		createAgentPackage(t, opampClient, "test-package-2", "AddonPackage", "2.0.0")

		packages := listAgentPackages(t, opampClient)
		assert.GreaterOrEqual(t, len(packages), 2, "Should have at least 2 packages")
	})

	// Test Get Agent Package
	t.Run("Get AgentPackage", func(t *testing.T) {
		pkg := getAgentPackage(t, opampClient, "test-package-1")
		assert.Equal(t, "test-package-1", pkg.Metadata.Name)
		assert.Equal(t, "TopLevelPackageName", pkg.Spec.PackageType)
	})

	// Test Update Agent Package
	t.Run("Update AgentPackage", func(t *testing.T) {
		pkg := getAgentPackage(t, opampClient, "test-package-1")
		pkg.Spec.Version = "1.1.0"
		pkg.Spec.DownloadURL = "https://example.com/updated-pkg.tar.gz"

		updated := updateAgentPackage(t, opampClient, pkg)
		assert.Equal(t, "1.1.0", updated.Spec.Version)
		assert.Equal(t, "https://example.com/updated-pkg.tar.gz", updated.Spec.DownloadURL)
	})

	// Test Delete Agent Package
	t.Run("Delete AgentPackage", func(t *testing.T) {
		deleteAgentPackage(t, opampClient, "test-package-2")

		// Verify deletion
		packages := listAgentPackages(t, opampClient)
		for _, pkg := range packages {
			assert.NotEqual(t, "test-package-2", pkg.Metadata.Name, "Deleted package should not exist")
		}
	})

	// Test Get Non-Existent Package
	t.Run("Get Non-Existent AgentPackage", func(t *testing.T) {
		_, err := opampClient.AgentPackageService.GetAgentPackage(t.Context(), "default", "non-existent-package")
		assert.Error(t, err, "Should return error for non-existent package")
	})
}

func createAgentPackage(t *testing.T, c *client.Client, name, packageType, version string) v1.AgentPackage {
	t.Helper()

	result, err := c.AgentPackageService.CreateAgentPackage(t.Context(), "default", &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       name,
			Attributes: v1.Attributes{"env": "test"},
		},
		Spec: v1.AgentPackageSpec{
			PackageType: packageType,
			Version:     version,
			DownloadURL: fmt.Sprintf("https://example.com/%s.tar.gz", name),
		},
	})
	require.NoError(t, err)

	return *result
}

func listAgentPackages(t *testing.T, c *client.Client) []v1.AgentPackage {
	t.Helper()

	resp, err := c.AgentPackageService.ListAgentPackages(t.Context(), "default")
	require.NoError(t, err)

	return resp.Items
}

func getAgentPackage(t *testing.T, c *client.Client, name string) v1.AgentPackage {
	t.Helper()

	result, err := c.AgentPackageService.GetAgentPackage(t.Context(), "default", name)
	require.NoError(t, err)

	return *result
}

func updateAgentPackage(t *testing.T, c *client.Client, pkg v1.AgentPackage) v1.AgentPackage {
	t.Helper()

	if pkg.Metadata.Namespace == "" {
		pkg.Metadata.Namespace = "default"
	}

	result, err := c.AgentPackageService.UpdateAgentPackage(t.Context(), &pkg)
	require.NoError(t, err)

	return *result
}

func deleteAgentPackage(t *testing.T, c *client.Client, name string) {
	t.Helper()

	err := c.AgentPackageService.DeleteAgentPackage(t.Context(), "default", name)
	require.NoError(t, err)
}

// TestE2E_AgentGroup_StatisticsAggregation tests that AgentGroup statistics aggregation
// works correctly using status.connected and status.componentHealth.healthy fields.
// This test verifies:
// - Aggregation uses indexed boolean fields instead of status.conditions
// - Statistics are correctly calculated for agents with various status field states
func TestE2E_AgentGroup_StatisticsAggregation(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()
	dbName := "opampcommander_e2e_stats_aggregation_test"

	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, dbName)
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: Insert agents directly into MongoDB with null status.conditions
	mongoClient, err := setupMongoDBClient(t, mongoURI)
	require.NoError(t, err)
	defer func() { _ = mongoClient.Disconnect(ctx) }()

	collection := mongoClient.Database(dbName).Collection("agents")

	agent1UID := uuid.New()
	agent2UID := uuid.New()
	agent3UID := uuid.New()

	// Insert agents with various status field states to test the aggregation:
	// - agent1: no connected/healthy fields (defaults to false)
	// - agent2: connected=false, componentHealth.healthy=false
	// - agent3: connected=true, componentHealth.healthy=true
	// Note: identifyingAttributes must be in KeyValuePairs format (array of {key, value})
	agentDocs := []interface{}{
		// Agent with no status fields set (defaults to not connected/healthy)
		bson.M{
			"metadata": bson.M{
				"instanceUid":       bson.Binary{Subtype: 0x04, Data: agent1UID[:]},
				"instanceUidString": agent1UID.String(),
				"description": bson.M{
					"identifyingAttributes": []bson.M{
						{"key": "service.name", "value": "stats-test-service"},
					},
				},
			},
			"status": bson.M{
				// connected and componentHealth are intentionally omitted
			},
			"spec": bson.M{},
		},
		// Agent with connected=false
		bson.M{
			"metadata": bson.M{
				"instanceUid":       bson.Binary{Subtype: 0x04, Data: agent2UID[:]},
				"instanceUidString": agent2UID.String(),
				"description": bson.M{
					"identifyingAttributes": []bson.M{
						{"key": "service.name", "value": "stats-test-service"},
					},
				},
			},
			"status": bson.M{
				"connected": false,
				"componentHealth": bson.M{
					"healthy": false,
				},
			},
			"spec": bson.M{},
		},
		// Agent with connected=true and healthy=true
		bson.M{
			"metadata": bson.M{
				"instanceUid":       bson.Binary{Subtype: 0x04, Data: agent3UID[:]},
				"instanceUidString": agent3UID.String(),
				"description": bson.M{
					"identifyingAttributes": []bson.M{
						{"key": "service.name", "value": "stats-test-service"},
					},
				},
			},
			"status": bson.M{
				"connected": true,
				"componentHealth": bson.M{
					"healthy": true,
				},
			},
			"spec": bson.M{},
		},
	}

	_, err = collection.InsertMany(ctx, agentDocs)
	require.NoError(t, err)

	t.Logf("Inserted 3 agents: no status fields, disconnected, connected+healthy")

	// When: Create an AgentGroup that matches these agents
	opampClient := createOpampClient(t, apiBaseURL)

	//exhaustruct:ignore
	agentGroup, err := opampClient.AgentGroupService.CreateAgentGroup(t.Context(), "default", &v1.AgentGroup{
		Metadata: v1.Metadata{
			Name: "stats-aggregation-group",
		},
		Spec: v1.Spec{
			Priority: 10,
			Selector: v1.AgentSelector{
				IdentifyingAttributes: map[string]string{
					"service.name": "stats-test-service",
				},
			},
		},
	})
	require.NoError(t, err, "AgentGroup creation should succeed")

	// Then: Statistics should be calculated correctly using status.connected and status.componentHealth.healthy
	assert.Equal(t, 3, agentGroup.Status.NumAgents, "Should count all 3 agents")
	assert.Equal(t, 1, agentGroup.Status.NumConnectedAgents, "Only agent3 has connected=true")
	assert.Equal(t, 1, agentGroup.Status.NumHealthyAgents, "Only agent3 has connected=true AND componentHealth.healthy=true")
	assert.Equal(t, 2, agentGroup.Status.NumNotConnectedAgents, "agent1 and agent2 are not connected")

	t.Logf("AgentGroup created with stats: NumAgents=%d, NumConnected=%d, NumHealthy=%d, NumNotConnected=%d",
		agentGroup.Status.NumAgents,
		agentGroup.Status.NumConnectedAgents,
		agentGroup.Status.NumHealthyAgents,
		agentGroup.Status.NumNotConnectedAgents)

	// When: Get the AgentGroup again (also triggers aggregation)
	got, err := opampClient.AgentGroupService.GetAgentGroup(t.Context(), "default", "stats-aggregation-group")
	require.NoError(t, err, "Get AgentGroup should succeed")
	assert.Equal(t, "stats-aggregation-group", got.Metadata.Name)

	// When: List AgentGroups (also triggers aggregation for each group)
	listed, err := opampClient.AgentGroupService.ListAgentGroups(t.Context(), "default")
	require.NoError(t, err, "List AgentGroups should succeed")
	assert.GreaterOrEqual(t, len(listed.Items), 1)
}

// TestE2E_AgentGroup_IncludeDeleted tests that deleted agent groups can be retrieved
// using the includeDeleted query parameter.
func TestE2E_AgentGroup_IncludeDeleted(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()
	dbName := "opampcommander_e2e_include_deleted_test"

	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, dbName)
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	opampClient := createOpampClient(t, apiBaseURL)

	// Step 1: Create an AgentGroup
	agentGroupName := "test-deleted-group"

	//exhaustruct:ignore
	_, err := opampClient.AgentGroupService.CreateAgentGroup(t.Context(), "default", &v1.AgentGroup{
		Metadata: v1.Metadata{
			Name: agentGroupName,
		},
		Spec: v1.Spec{
			Selector: v1.AgentSelector{
				IdentifyingAttributes: map[string]string{"service.name": "test-service"},
			},
		},
	})
	require.NoError(t, err, "Create AgentGroup should succeed")

	// Step 2: Verify the AgentGroup is in the list
	listResp, err := opampClient.AgentGroupService.ListAgentGroups(t.Context(), "default")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(listResp.Items), 1, "Should have at least 1 agent group")

	found := false
	for _, ag := range listResp.Items {
		if ag.Metadata.Name == agentGroupName {
			found = true
			break
		}
	}
	require.True(t, found, "Created agent group should be in the list")

	// Step 3: Delete the AgentGroup
	err = opampClient.AgentGroupService.DeleteAgentGroup(t.Context(), "default", agentGroupName)
	require.NoError(t, err, "Delete AgentGroup should succeed")

	// Step 4: Verify the AgentGroup is NOT in the regular list
	listResp, err = opampClient.AgentGroupService.ListAgentGroups(t.Context(), "default")
	require.NoError(t, err)

	found = false
	for _, ag := range listResp.Items {
		if ag.Metadata.Name == agentGroupName {
			found = true
			break
		}
	}
	require.False(t, found, "Deleted agent group should NOT be in the regular list")

	// Step 5: Verify the AgentGroup IS in the list when using includeDeleted=true
	listResp, err = opampClient.AgentGroupService.ListAgentGroups(t.Context(), "default",
		client.WithIncludeDeleted(true))
	require.NoError(t, err)

	found = false
	for _, ag := range listResp.Items {
		if ag.Metadata.Name == agentGroupName {
			found = true
			// Verify it has DeletedAt set
			require.NotNil(t, ag.Metadata.DeletedAt, "Deleted agent group should have DeletedAt")
			break
		}
	}
	require.True(t, found, "Deleted agent group should be in the list when includeDeleted=true")

	// Step 6: Verify the deleted AgentGroup can be retrieved by name with includeDeleted=true
	ag, err := opampClient.AgentGroupService.GetAgentGroup(t.Context(), "default", agentGroupName,
		client.WithGetIncludeDeleted(true))
	require.NoError(t, err, "Get deleted AgentGroup with includeDeleted=true should succeed")
	require.Equal(t, agentGroupName, ag.Metadata.Name)
	require.NotNil(t, ag.Metadata.DeletedAt, "Retrieved deleted agent group should have DeletedAt")

	// Step 7: Verify the deleted AgentGroup returns error without includeDeleted
	_, err = opampClient.AgentGroupService.GetAgentGroup(t.Context(), "default", agentGroupName)
	require.Error(t, err, "Get deleted AgentGroup without includeDeleted should return error")
}

