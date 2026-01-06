//go:build e2e

package apiserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	testMongoDBImage      = "mongo:4.4.10"
	otelCollectorImage    = "otel/opentelemetry-collector-contrib:0.115.1"
	apiServerStartTimeout = 15 * time.Second
)

func TestE2E_APIServer_WithOTelCollector(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	mongoContainer, mongoURI := startMongoDB(t)

	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()

	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_test")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: OTel Collector is started
	collectorUID := uuid.New()
	collectorCfg := createCollectorConfig(t, base.CacheDir, apiPort, collectorUID)
	collectorContainer := startOTelCollector(t, collectorCfg)

	defer func() { _ = collectorContainer.Terminate(ctx) }()

	// When: Collector reports via OpAMP
	assert.Eventually(t, func() bool {
		agents := listAgents(t, apiBaseURL)

		return len(agents) > 0
	}, 3*time.Minute, 1*time.Second, "At least one agent should register within timeout")

	assert.Eventually(t, func() bool {
		// Then: Collector has complete metadata
		agents := listAgents(t, apiBaseURL)
		agent := findAgentByUID(agents, collectorUID)
		require.NotNil(t, agent, "Collector should be registered")

		hasDescription := len(agent.Metadata.Description.IdentifyingAttributes) > 0 ||
			len(agent.Metadata.Description.NonIdentifyingAttributes) > 0
		if !hasDescription {
			return false // Agent does not have description yet
		}

		hasCapabilities := agent.Metadata.Capabilities != 0

		return hasCapabilities // Agent should have capabilities
	}, 30*time.Second, 1*time.Second, "Agent metadata should be complete within timeout")

	// Then: Agent is retrievable by ID
	assert.Eventually(t, func() bool {
		specificAgent := getAgentByID(t, apiBaseURL, collectorUID)

		return specificAgent.Metadata.InstanceUID == collectorUID
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
	collectorUIDs := make([]uuid.UUID, numCollectors)
	collectors := make([]testcontainers.Container, numCollectors)

	for i := range numCollectors {
		collectorUIDs[i] = uuid.New()
		cfg := createCollectorConfig(t, base.CacheDir, apiPort, collectorUIDs[i])
		collector := startOTelCollector(t, cfg)
		collectors[i] = collector
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

		for _, uid := range collectorUIDs {
			if findAgentByUID(agents, uid) != nil {
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

func setupAPIServer(t *testing.T, port int, mongoURI, dbName string) (func(), string) {
	t.Helper()

	hostname, _ := os.Hostname()
	serverID := fmt.Sprintf("%s-test-%d", hostname, port)

	managementPort, err := testutil.GetFreeTCPPort()
	require.NoError(t, err)

	//exhaustruct:ignore
	settings := config.ServerSettings{
		Address:  fmt.Sprintf("0.0.0.0:%d", port),
		ServerID: config.ServerID(serverID),
		EventSettings: config.EventSettings{
			ProtocolType: config.EventProtocolTypeInMemory,
		},
		DatabaseSettings: config.DatabaseSettings{
			Type:           config.DatabaseTypeMongoDB,
			Endpoints:      []string{mongoURI},
			ConnectTimeout: 10 * time.Second,
			DatabaseName:   dbName,
			DDLAuto:        true,
		},
		//exhaustruct:ignore
		AuthSettings: config.AuthSettings{
			//exhaustruct:ignore
			AdminSettings: config.AdminSettings{
				Username: "test-admin",
				Password: "test-password",
				Email:    "test@test.com",
			},
			//exhaustruct:ignore
			JWTSettings: config.JWTSettings{
				SigningKey: "test-secret-key",
				Issuer:     "e2e-test",
				Expiration: 24 * time.Hour,
				Audience:   []string{"test"},
			},
		},
		//exhaustruct:ignore
		ManagementSettings: config.ManagementSettings{
			Address: fmt.Sprintf(":%d", managementPort),
			//exhaustruct:ignore
			ObservabilitySettings: config.ObservabilitySettings{
				//exhaustruct:ignore
				Log: config.LogSettings{
					Format: config.LogFormatText,
				},
			},
		},
	}

	server := apiserver.New(settings)
	serverCtx, cancel := context.WithCancel(t.Context())

	go func() {
		_ = server.Run(serverCtx)
	}()

	stopServer := func() {
		cancel()
	}

	apiBaseURL := fmt.Sprintf("http://localhost:%d", port)

	return stopServer, apiBaseURL
}

func getAuthToken(t *testing.T, baseURL string) string {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/auth/basic", nil) //nolint:noctx
	require.NoError(t, err)
	req.SetBasicAuth("test-admin", "test-password")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(body, &result))

	return result.Token
}

func getAuthTokenNoTest(baseURL string) string {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/auth/basic", nil) //nolint:noctx
	if err != nil {
		return ""
	}
	req.SetBasicAuth("test-admin", "test-password")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var result struct {
		Token string `json:"token"`
	}
	if json.Unmarshal(body, &result) != nil {
		return ""
	}

	return result.Token
}

func waitForAPIServerReady(t *testing.T, baseURL string) {
	t.Helper()

	require.Eventually(t, func() bool {
		resp, err := http.Get(baseURL + "/api/v1/ping") //nolint:noctx // test helper
		if err != nil {
			return false
		}

		defer func() { _ = resp.Body.Close() }()

		return resp.StatusCode == http.StatusOK
	}, apiServerStartTimeout, 500*time.Millisecond, "API server should start")
}

func listAgents(t *testing.T, baseURL string) []v1agent.Agent {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/agents", nil) //nolint:noctx
	require.NoError(t, err)

	token := getAuthToken(t, baseURL)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result struct {
		Items []v1agent.Agent `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &result))

	return result.Items
}

func getAgentByID(t *testing.T, baseURL string, uid uuid.UUID) v1agent.Agent {
	t.Helper()

	agent, err := tryGetAgentByID(baseURL, uid)
	require.NoError(t, err)

	return agent
}

func tryGetAgentByID(baseURL string, uid uuid.UUID) (v1agent.Agent, error) {
	url := fmt.Sprintf("%s/api/v1/agents/%s", baseURL, uid)

	req, err := http.NewRequest(http.MethodGet, url, nil) //nolint:noctx
	if err != nil {
		return v1agent.Agent{}, fmt.Errorf("failed to create request: %w", err)
	}

	token := getAuthTokenNoTest(baseURL)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return v1agent.Agent{}, fmt.Errorf("failed to get agent by ID: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return v1agent.Agent{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode) //nolint:err113
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return v1agent.Agent{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var agent v1agent.Agent

	err = json.Unmarshal(body, &agent)
	if err != nil {
		return v1agent.Agent{}, fmt.Errorf("failed to unmarshal agent: %w", err)
	}

	return agent, nil
}

func findAgentByUID(agents []v1agent.Agent, uid uuid.UUID) *v1agent.Agent {
	for i := range agents {
		if agents[i].Metadata.InstanceUID == uid {
			return &agents[i]
		}
	}

	return nil
}

func createCollectorConfig(t *testing.T, cacheDir string, opampPort int, instanceUID uuid.UUID) string {
	t.Helper()

	configContent := fmt.Sprintf(`receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:

exporters:
  debug:
    verbosity: basic

extensions:
  opamp:
    server:
      ws:
        endpoint: ws://host.docker.internal:%d/api/v1/opamp
        tls:
          insecure: true
        headers:
          X-Test-Header: e2e-test
    instance_uid: %s

service:
  extensions: [opamp]
  telemetry:
    logs:
      level: info
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
`, opampPort, instanceUID.String())

	configPath := filepath.Join(cacheDir, fmt.Sprintf("collector-config-%s.yaml", instanceUID.String()))
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	return configPath
}

func startOTelCollector(t *testing.T, configPath string) testcontainers.Container {
	t.Helper()

	//exhaustruct:ignore
	req := testcontainers.ContainerRequest{
		Image:        otelCollectorImage,
		ExposedPorts: []string{"4317/tcp", "4318/tcp"},
		Files: []testcontainers.ContainerFile{
			//exhaustruct:ignore
			{
				HostFilePath:      configPath,
				ContainerFilePath: "/etc/otel-collector-config.yaml",
				FileMode:          0644,
			},
		},
		Cmd:        []string{"--config=/etc/otel-collector-config.yaml"},
		WaitingFor: wait.ForLog("Everything is ready").WithStartupTimeout(60 * time.Second),
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
	}

	//exhaustruct:ignore
	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	return container
}

func TestE2E_APIServer_SequenceNum(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	mongoContainer, mongoURI := startMongoDB(t)

	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()
	dbName := "opampcommander_e2e_sequencenum_test"

	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, dbName)
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Setup MongoDB client to verify data directly
	mongoClient, err := setupMongoDBClient(t, mongoURI)
	require.NoError(t, err)

	defer func() { _ = mongoClient.Disconnect(ctx) }()

	// Given: OTel Collector is started
	collectorUID := uuid.New()
	collectorCfg := createCollectorConfig(t, base.CacheDir, apiPort, collectorUID)
	collectorContainer := startOTelCollector(t, collectorCfg)

	defer func() { _ = collectorContainer.Terminate(ctx) }()

	// When: Collector reports via OpAMP multiple times
	t.Log("Waiting for collector to register...")
	require.Eventually(t, func() bool {
		agents := listAgents(t, apiBaseURL)

		return len(agents) > 0
	}, 2*time.Minute, 1*time.Second, "Agent should register within timeout")

	// Then: SequenceNum should be visible through API and incrementing
	t.Log("Verifying SequenceNum through API...")

	var previousSeqNum uint64

	require.Eventually(t, func() bool {
		agent := getAgentByID(t, apiBaseURL, collectorUID)

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
	verifySequenceNumInMongoDB(t, mongoClient, dbName, collectorUID)

	// Then: Final verification - get multiple reports and ensure monotonic increase
	t.Log("Final verification of SequenceNum progression...")

	seqNums := make([]uint64, 0, 5)

	for i := range 5 {
		time.Sleep(3 * time.Second)

		agent := getAgentByID(t, apiBaseURL, collectorUID)
		seqNums = append(seqNums, agent.Status.SequenceNum)
		t.Logf("Sample %d: SequenceNum = %d", i+1, agent.Status.SequenceNum)
	}

	// Verify monotonic increase
	for i := 1; i < len(seqNums); i++ {
		assert.GreaterOrEqual(t, seqNums[i], seqNums[i-1],
			"SequenceNum should be monotonically increasing or equal")
	}

	// At least some should have increased
	assert.Greater(t, seqNums[len(seqNums)-1], seqNums[0],
		"SequenceNum should have increased over time")
}

func setupMongoDBClient(t *testing.T, mongoURI string) (*mongo.Client, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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
	searchResp := searchAgents(t, apiBaseURL, "12345678-1234")

	// Then: Should find only agent1
	assert.Len(t, searchResp.Items, 1)
	assert.Equal(t, agent1UID, searchResp.Items[0].Metadata.InstanceUID)

	// When: Search by prefix "1234"
	t.Log("Searching for agents with prefix '1234'...")
	searchResp = searchAgents(t, apiBaseURL, "1234")

	// Then: Should find agent1 and agent2
	assert.Len(t, searchResp.Items, 2)

	// When: Search by prefix "abcd" (case-insensitive)
	t.Log("Searching for agents with prefix 'ABCD' (uppercase)...")
	searchResp = searchAgents(t, apiBaseURL, "ABCD")

	// Then: Should find agent3
	assert.Len(t, searchResp.Items, 1)
	assert.Equal(t, agent3UID, searchResp.Items[0].Metadata.InstanceUID)

	// When: Search with no matches
	t.Log("Searching for agents with non-existent prefix...")
	searchResp = searchAgents(t, apiBaseURL, "zzzzz")

	// Then: Should return empty result
	assert.Empty(t, searchResp.Items)

	// When: Search with pagination
	t.Log("Searching with pagination (limit=1)...")
	searchResp = searchAgentsWithLimit(t, apiBaseURL, "1234", 1)

	// Then: Should return 1 result and have continue token
	assert.Len(t, searchResp.Items, 1)
	assert.NotEmpty(t, searchResp.Metadata.Continue)
}

func searchAgents(t *testing.T, apiBaseURL, query string) *v1agent.ListResponse {
	t.Helper()

	return searchAgentsWithLimit(t, apiBaseURL, query, 0)
}

func searchAgentsWithLimit(t *testing.T, apiBaseURL, query string, limit int) *v1agent.ListResponse {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/agents/search?q=%s", apiBaseURL, query)
	if limit > 0 {
		url = fmt.Sprintf("%s&limit=%d", url, limit)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)

	token := getAuthToken(t, apiBaseURL)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Search request should succeed")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var listResp v1agent.ListResponse

	err = json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	return &listResp
}

// TestE2E_ConnectionType_HTTPAndWebSocket tests that HTTP and WebSocket connections
// are correctly identified and reported via the connections API.
func TestE2E_ConnectionType_HTTPAndWebSocket(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()
	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_conntype_test")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: Two OTel Collectors - one with HTTP polling, one with WebSocket
	httpCollectorUID := uuid.New()
	wsCollectorUID := uuid.New()

	// Start HTTP polling collector
	httpCollectorCfg := createCollectorConfigWithProtocol(t, base.CacheDir, apiPort, httpCollectorUID, "http")
	httpCollectorContainer := startOTelCollector(t, httpCollectorCfg)
	defer func() { _ = httpCollectorContainer.Terminate(ctx) }()

	// Start WebSocket collector
	wsCollectorCfg := createCollectorConfigWithProtocol(t, base.CacheDir, apiPort, wsCollectorUID, "ws")
	wsCollectorContainer := startOTelCollector(t, wsCollectorCfg)
	defer func() { _ = wsCollectorContainer.Terminate(ctx) }()

	// When: Both collectors connect
	assert.Eventually(t, func() bool {
		agents := listAgents(t, apiBaseURL)
		return len(agents) >= 2
	}, 2*time.Minute, 1*time.Second, "Both collectors should register")

	// Then: Verify agents are registered
	agents := listAgents(t, apiBaseURL)
	require.GreaterOrEqual(t, len(agents), 2, "Both collectors should register as agents")

	// Find our collectors in the agents list
	httpAgent := findAgentByUID(agents, httpCollectorUID)
	wsAgent := findAgentByUID(agents, wsCollectorUID)
	require.NotNil(t, httpAgent, "HTTP collector should be registered as agent")
	require.NotNil(t, wsAgent, "WebSocket collector should be registered as agent")

	// Then: Check that WebSocket connection is active
	// Note: HTTP polling connections are not persistent, so they won't appear in connections list
	assert.Eventually(t, func() bool {
		connections := listConnections(t, apiBaseURL)
		t.Logf("Found %d active connections", len(connections))
		
		for _, conn := range connections {
			t.Logf("Connection InstanceUID: %s, Type: %s", conn.InstanceUID, conn.Type)
			if conn.InstanceUID == wsCollectorUID && conn.Type == "WebSocket" {
				t.Logf("WebSocket collector connection found")
				return true
			}
		}
		
		return false
	}, 30*time.Second, 1*time.Second, "WebSocket connection should be active")
}

// listConnections retrieves all connections from the API.
func listConnections(t *testing.T, baseURL string) []connectionResponse {
	t.Helper()

	token := getAuthToken(t, baseURL)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/connections", nil) //nolint:noctx
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result struct {
		Items []connectionResponse `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &result))

	return result.Items
}

// connectionResponse represents a connection from the API response.
type connectionResponse struct {
	ID                 uuid.UUID `json:"id"`
	InstanceUID        uuid.UUID `json:"instanceUid"`
	Type               string    `json:"type"`
	LastCommunicatedAt time.Time `json:"lastCommunicatedAt"`
	Alive              bool      `json:"alive"`
}

// createCollectorConfigWithProtocol creates a collector config with specified protocol.
func createCollectorConfigWithProtocol(t *testing.T, cacheDir string, apiPort int, collectorUID uuid.UUID, protocol string) string {
	t.Helper()

	var configContent string
	if protocol == "http" {
		// HTTP polling configuration
		configContent = fmt.Sprintf(`receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  batch:

exporters:
  debug:
    verbosity: detailed

extensions:
  opamp:
    server:
      http:
        endpoint: http://host.docker.internal:%d/api/v1/opamp
        headers:
          User-Agent: "e2e-test-collector-http"
    instance_uid: %s

service:
  extensions: [opamp]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
`, apiPort, collectorUID.String())
	} else {
		// WebSocket configuration
		configContent = fmt.Sprintf(`receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  batch:

exporters:
  debug:
    verbosity: detailed

extensions:
  opamp:
    server:
      ws:
        endpoint: ws://host.docker.internal:%d/api/v1/opamp
        tls:
          insecure: true
        headers:
          User-Agent: "e2e-test-collector-ws"
    instance_uid: %s

service:
  extensions: [opamp]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
`, apiPort, collectorUID.String())
	}

	configPath := filepath.Join(cacheDir, fmt.Sprintf("collector-config-%s-%s.yaml", protocol, collectorUID.String()))
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	return configPath
}
