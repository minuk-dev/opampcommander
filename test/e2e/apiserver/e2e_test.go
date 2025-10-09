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

func TestE2E_APIServer_WithOTelCollector(t *testing.T) {
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

	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_test")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: OTel Collector is started
	collectorUID := uuid.New()
	collectorCfg := createCollectorConfig(t, base.CacheDir, apiPort, collectorUID)
	collectorContainer := startOTelCollector(t, collectorCfg)

	defer func() { _ = collectorContainer.Terminate(ctx) }()

	// When: Collector reports via OpAMP
	time.Sleep(5 * time.Second)

	// Then: Agent is registered
	agents := listAgents(t, apiBaseURL)
	require.GreaterOrEqual(t, len(agents), 1, "At least one agent should be registered")

	// Then: Collector has complete metadata
	agent := findAgentByUID(agents, collectorUID)
	require.NotNil(t, agent, "Collector should be registered")
	assertAgentMetadataComplete(t, agent)

	// Then: Agent is retrievable by ID
	specificAgent := getAgentByID(t, apiBaseURL, collectorUID)
	assert.Equal(t, collectorUID, specificAgent.InstanceUID)
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
	time.Sleep(8 * time.Second)

	// Then: All agents are registered
	agents := listAgents(t, apiBaseURL)
	assert.GreaterOrEqual(t, len(agents), numCollectors, "All collectors should be registered")

	// Then: Each collector is found
	foundCount := 0

	for _, uid := range collectorUIDs {
		if findAgentByUID(agents, uid) != nil {
			foundCount++
		}
	}

	assert.Equal(t, numCollectors, foundCount, "All collectors should be found")
}

// Helper functions for Given-When-Then pattern

//nolint:ireturn
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

	//exhaustruct:ignore
	settings := config.ServerSettings{
		Address: fmt.Sprintf("0.0.0.0:%d", port),
		DatabaseSettings: config.DatabaseSettings{
			Type:           config.DatabaseTypeMongoDB,
			Endpoints:      []string{mongoURI},
			ConnectTimeout: 10 * time.Second,
			DatabaseName:   dbName,
		},
		//exhaustruct:ignore
		AuthSettings: config.AuthSettings{
			//exhaustruct:ignore
			JWTSettings: config.JWTSettings{
				SigningKey: "test-secret-key",
				Issuer:     "e2e-test",
				Expiration: 24 * time.Hour,
				Audience:   []string{"test"},
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

func waitForAPIServerReady(t *testing.T, baseURL string) {
	t.Helper()

	require.Eventually(t, func() bool {
		resp, err := http.Get(baseURL + "/api/v1/health") //nolint:noctx // test helper
		if err != nil {
			return false
		}

		defer func() { _ = resp.Body.Close() }()

		return resp.StatusCode == http.StatusOK
	}, apiServerStartTimeout, 500*time.Millisecond, "API server should start")
}

func listAgents(t *testing.T, baseURL string) []v1agent.Agent {
	t.Helper()

	resp, err := http.Get(baseURL + "/api/v1/agents") //nolint:noctx // test helper
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

	url := fmt.Sprintf("%s/api/v1/agents/%s", baseURL, uid)
	resp, err := http.Get(url) //nolint:noctx,gosec // test helper
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var agent v1agent.Agent
	require.NoError(t, json.Unmarshal(body, &agent))

	return agent
}

func findAgentByUID(agents []v1agent.Agent, uid uuid.UUID) *v1agent.Agent {
	for i := range agents {
		if agents[i].InstanceUID == uid {
			return &agents[i]
		}
	}

	return nil
}

func assertAgentMetadataComplete(t *testing.T, agent *v1agent.Agent) {
	t.Helper()

	hasDescription := len(agent.Description.IdentifyingAttributes) > 0 ||
		len(agent.Description.NonIdentifyingAttributes) > 0
	assert.True(t, hasDescription, "Agent should have description")

	hasCapabilities := agent.Capabilities != 0
	assert.True(t, hasCapabilities, "Agent should have capabilities")
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
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	return configPath
}

//nolint:ireturn
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
	}

	//exhaustruct:ignore
	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	return container
}
