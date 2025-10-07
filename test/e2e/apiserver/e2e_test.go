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

// TestE2E_APIServer_WithOTelCollector tests the complete flow:
// 1. Start MongoDB
// 2. Start API Server
// 3. Start OTel Collector with OpAMP extension
// 4. Verify collector connects via OpAMP
// 5. Query Agent API and verify response
func TestE2E_APIServer_WithOTelCollector(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Start MongoDB
	mongoContainer, mongoURI := startMongoDB(t, ctx)
	defer func() {
		_ = mongoContainer.Terminate(ctx)
	}()

	// Start API Server
	apiPort := base.GetFreeTCPPort()

	serverSettings := config.ServerSettings{
		Address: fmt.Sprintf("0.0.0.0:%d", apiPort),
		DatabaseSettings: config.DatabaseSettings{
			Type:           config.DatabaseTypeMongoDB,
			Endpoints:      []string{mongoURI},
			ConnectTimeout: 10 * time.Second,
			DatabaseName:   "opampcommander_e2e_test",
		},
		AuthSettings: config.AuthSettings{
			JWTSettings: config.JWTSettings{
				SigningKey: "test-secret-key-for-e2e-testing",
				Issuer:     "opampcommander-e2e-test",
				Expiration: 24 * time.Hour,
				Audience:   []string{"opampcommander"},
			},
		},
	}

	server := apiserver.New(serverSettings)

	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Run(serverCtx)
	}()

	// Wait for server to be ready
	apiBaseURL := fmt.Sprintf("http://localhost:%d", apiPort)
	require.Eventually(t, func() bool {
		resp, err := http.Get(apiBaseURL + "/api/v1/health")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, apiServerStartTimeout, 500*time.Millisecond, "API server should start within timeout")

	// Start OTel Collector with OpAMP extension
	collectorInstanceUID := uuid.New()
	collectorConfig := createCollectorConfig(t, base.CacheDir, apiPort, collectorInstanceUID)

	collectorContainer, _ := startOTelCollector(t, ctx, collectorConfig)
	defer func() {
		_ = collectorContainer.Terminate(ctx)
	}()

	// Wait for collector to connect via OpAMP
	time.Sleep(5 * time.Second)

	// Query Agent API and verify response
	agentsResp, err := http.Get(apiBaseURL + "/api/v1/agents")
	require.NoError(t, err, "Should be able to call agents API")
	defer agentsResp.Body.Close()

	assert.Equal(t, http.StatusOK, agentsResp.StatusCode, "Agents API should return 200 OK")

	agentsBody, err := io.ReadAll(agentsResp.Body)
	require.NoError(t, err)

	var agentsList struct {
		Items []v1agent.Agent `json:"items"`
	}
	err = json.Unmarshal(agentsBody, &agentsList)
	require.NoError(t, err, "Should be able to parse agents response")

	// Verify at least one agent exists
	require.GreaterOrEqual(t, len(agentsList.Items), 1, "At least one agent should be registered")

	// Find our collector
	var collectorAgent *v1agent.Agent
	for i := range agentsList.Items {
		if agentsList.Items[i].InstanceUID.String() == collectorInstanceUID.String() {
			collectorAgent = &agentsList.Items[i]
			break
		}
	}

	require.NotNil(t, collectorAgent, "Collector agent should be found in the list")

	// Verify agent details
	t.Log("Step 6: Verifying agent details...")
	assert.Equal(t, collectorInstanceUID, collectorAgent.InstanceUID, "Instance UID should match")

	// Check metadata completeness
	hasDescription := len(collectorAgent.Description.IdentifyingAttributes) > 0 ||
		len(collectorAgent.Description.NonIdentifyingAttributes) > 0
	assert.True(t, hasDescription, "Agent should have description")

	hasCapabilities := collectorAgent.Capabilities != 0
	assert.True(t, hasCapabilities, "Agent should have capabilities")

	// Get specific agent by ID
	agentURL := fmt.Sprintf("%s/api/v1/agents/%s", apiBaseURL, collectorInstanceUID)
	agentResp, err := http.Get(agentURL)
	require.NoError(t, err, "Should be able to get specific agent")
	defer agentResp.Body.Close()

	assert.Equal(t, http.StatusOK, agentResp.StatusCode, "Get agent API should return 200 OK")

	agentBody, err := io.ReadAll(agentResp.Body)
	require.NoError(t, err)

	var specificAgent v1agent.Agent
	err = json.Unmarshal(agentBody, &specificAgent)
	require.NoError(t, err, "Should be able to parse agent response")

	assert.Equal(t, collectorInstanceUID, specificAgent.InstanceUID, "Instance UID should match")
}

// TestE2E_APIServer_MultipleCollectors tests multiple collectors connecting simultaneously.
func TestE2E_APIServer_MultipleCollectors(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Start MongoDB
	mongoContainer, mongoURI := startMongoDB(t, ctx)
	defer func() {
		_ = mongoContainer.Terminate(ctx)
	}()

	// Start API Server
	apiPort := base.GetFreeTCPPort()

	serverSettings := config.ServerSettings{
		Address: fmt.Sprintf("0.0.0.0:%d", apiPort),
		DatabaseSettings: config.DatabaseSettings{
			Type:           config.DatabaseTypeMongoDB,
			Endpoints:      []string{mongoURI},
			ConnectTimeout: 10 * time.Second,
			DatabaseName:   "opampcommander_e2e_test_multi",
		},
		AuthSettings: config.AuthSettings{
			JWTSettings: config.JWTSettings{
				SigningKey: "test-secret-key-for-e2e-testing",
				Issuer:     "opampcommander-e2e-test",
				Expiration: 24 * time.Hour,
				Audience:   []string{"opampcommander"},
			},
		},
	}

	server := apiserver.New(serverSettings)

	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	go func() {
		_ = server.Run(serverCtx)
	}()

	apiBaseURL := fmt.Sprintf("http://localhost:%d", apiPort)
	require.Eventually(t, func() bool {
		resp, err := http.Get(apiBaseURL + "/api/v1/health")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, apiServerStartTimeout, 500*time.Millisecond)

	// Start multiple collectors
	numCollectors := 3
	collectorUIDs := make([]uuid.UUID, numCollectors)
	collectors := make([]testcontainers.Container, numCollectors)

	for i := 0; i < numCollectors; i++ {
		collectorUIDs[i] = uuid.New()
		collectorConfig := createCollectorConfig(t, base.CacheDir, apiPort, collectorUIDs[i])

		collector, _ := startOTelCollector(t, ctx, collectorConfig)
		collectors[i] = collector
	}

	// Cleanup collectors
	defer func() {
		for _, c := range collectors {
			_ = c.Terminate(ctx)
		}
	}()

	// Wait for all collectors to connect
	time.Sleep(8 * time.Second)

	// Verify all agents are registered
	agentsResp, err := http.Get(apiBaseURL + "/api/v1/agents")
	require.NoError(t, err)
	defer agentsResp.Body.Close()

	agentsBody, err := io.ReadAll(agentsResp.Body)
	require.NoError(t, err)

	var agentsList struct {
		Items []v1agent.Agent `json:"items"`
	}
	err = json.Unmarshal(agentsBody, &agentsList)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(agentsList.Items), numCollectors,
		"Should have at least %d agents", numCollectors)

	// Verify each collector is registered
	foundCount := 0
	for _, uid := range collectorUIDs {
		for _, agent := range agentsList.Items {
			if agent.InstanceUID == uid {
				foundCount++
				break
			}
		}
	}

	assert.Equal(t, numCollectors, foundCount, "All collectors should be registered")
}

// startMongoDB starts a MongoDB container and returns the container and connection URI.
func startMongoDB(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
	t.Helper()

	container, err := mongoTestContainer.Run(ctx, testMongoDBImage)
	require.NoError(t, err, "Should start MongoDB container")

	uri, err := container.ConnectionString(ctx)
	require.NoError(t, err, "Should get MongoDB connection string")

	return container, uri
}

// createCollectorConfig creates an OTel Collector configuration file with OpAMP extension.
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
        headers:
          X-Test-Header: e2e-test
    instance_uid: %s
    capabilities:
      reports_effective_config: true
      reports_health: true
      reports_available_components: true
    agent_description:
      non_identifying_attributes:
        service.name: otel-collector-e2e-test
        service.instance.id: %s
        os.type: linux
        host.name: collector-test-host

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
`, opampPort, instanceUID.String(), instanceUID.String())

	configPath := filepath.Join(cacheDir, fmt.Sprintf("collector-config-%s.yaml", instanceUID.String()))
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err, "Should write collector config file")

	return configPath
}

// startOTelCollector starts an OTel Collector container with the given configuration.
func startOTelCollector(t *testing.T, ctx context.Context, configPath string) (testcontainers.Container, string) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        otelCollectorImage,
		ExposedPorts: []string{"4317/tcp", "4318/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      configPath,
				ContainerFilePath: "/etc/otel-collector-config.yaml",
				FileMode:          0644,
			},
		},
		Cmd: []string{"--config=/etc/otel-collector-config.yaml"},
		WaitingFor: wait.ForLog("Everything is ready").
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Should start OTel Collector container")

	// Get container name or ID for logging
	name, err := container.Name(ctx)
	containerID := name
	if err != nil {
		containerID = "unknown"
	}

	return container, containerID
}
