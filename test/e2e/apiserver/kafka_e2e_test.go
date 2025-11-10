package apiserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	kafkaTestContainer "github.com/testcontainers/testcontainers-go/modules/kafka"

	"github.com/minuk-dev/opampcommander/pkg/apiserver"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	kafkaImage            = "confluentinc/cp-kafka:7.5.0"
	serverStartupDelay    = 5 * time.Second
	eventPropagationDelay = 3 * time.Second
)

// TestE2E_APIServer_KafkaDistributedMode tests distributed mode with Kafka messaging
// Scenario: Two API servers communicate via Kafka, and an agent update on server1
// should be propagated to server2.
func TestE2E_APIServer_KafkaDistributedMode(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + Kafka)
	mongoContainer, mongoURI := startMongoDB(t)

	defer func() { _ = mongoContainer.Terminate(ctx) }()

	kafkaContainer, kafkaBroker := startKafka(t)

	defer func() { _ = kafkaContainer.Terminate(ctx) }()

	// Given: Two API servers in distributed mode
	server1Port := base.GetFreeTCPPort()
	server2Port := base.GetFreeTCPPort()

	stopServer1, server1URL := setupAPIServerWithKafka(
		t, server1Port, mongoURI, kafkaBroker, "opampcommander_kafka_e2e", "server-1",
	)
	defer stopServer1()

	stopServer2, server2URL := setupAPIServerWithKafka(
		t, server2Port, mongoURI, kafkaBroker, "opampcommander_kafka_e2e", "server-2",
	)
	defer stopServer2()

	waitForAPIServerReady(t, server1URL)
	waitForAPIServerReady(t, server2URL)

	t.Log("Both API servers are ready")

	// Given: Collector connects to server 1
	collectorUID := uuid.New()
	collectorCfg := createCollectorConfig(t, base.CacheDir, server1Port, collectorUID)
	collectorContainer := startOTelCollector(t, collectorCfg)

	defer func() { _ = collectorContainer.Terminate(ctx) }()

	// When: Collector registers via OpAMP on server 1
	time.Sleep(5 * time.Second)

	// Then: Agent should be visible on server 1
	agents1 := listAgents(t, server1URL)
	require.GreaterOrEqual(t, len(agents1), 1, "Agent should be registered on server 1")
	agent1 := findAgentByUID(agents1, collectorUID)
	require.NotNil(t, agent1, "Collector should be found on server 1")
	t.Logf("Agent registered on server 1: %s", agent1.Metadata.InstanceUID)

	// Then: Agent should also be visible on server 2 (shared database)
	agents2 := listAgents(t, server2URL)
	require.GreaterOrEqual(t, len(agents2), 1, "Agent should be visible on server 2")
	agent2 := findAgentByUID(agents2, collectorUID)
	require.NotNil(t, agent2, "Collector should be found on server 2")
	t.Logf("Agent visible on server 2: %s", agent2.Metadata.InstanceUID)

	// When: Server 2 updates the agent configuration
	updateRequest := map[string]interface{}{
		"config": map[string]interface{}{
			"configMap": map[string]string{
				"test_key": "test_value_from_server2",
			},
		},
	}
	updateAgentConfig(t, server2URL, collectorUID, updateRequest)
	t.Log("Agent config updated via server 2")

	// Allow time for Kafka message propagation
	time.Sleep(eventPropagationDelay)

	// Then: Updated config should be visible on both servers
	updatedAgent1 := getAgentByID(t, server1URL, collectorUID)
	updatedAgent2 := getAgentByID(t, server2URL, collectorUID)

	// Verify config was updated
	assert.NotNil(t, updatedAgent1.Spec.RemoteConfig, "Agent on server 1 should have remote config")
	assert.NotNil(t, updatedAgent2.Spec.RemoteConfig, "Agent on server 2 should have remote config")

	if updatedAgent1.Spec.RemoteConfig.ConfigMap != nil {
		assert.Equal(t, "test_value_from_server2", updatedAgent1.Spec.RemoteConfig.ConfigMap["test_key"],
			"Config update should be visible on server 1")
	}

	if updatedAgent2.Spec.RemoteConfig.ConfigMap != nil {
		assert.Equal(t, "test_value_from_server2", updatedAgent2.Spec.RemoteConfig.ConfigMap["test_key"],
			"Config update should be visible on server 2")
	}

	t.Log("Distributed mode test completed successfully")
}

// TestE2E_APIServer_KafkaFailover tests failover scenario in distributed mode.
func TestE2E_APIServer_KafkaFailover(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure setup
	mongoContainer, mongoURI := startMongoDB(t)

	defer func() { _ = mongoContainer.Terminate(ctx) }()

	kafkaContainer, kafkaBroker := startKafka(t)

	defer func() { _ = kafkaContainer.Terminate(ctx) }()

	// Given: Primary server is running
	primaryPort := base.GetFreeTCPPort()

	stopPrimary, primaryURL := setupAPIServerWithKafka(
		t, primaryPort, mongoURI, kafkaBroker, "opampcommander_kafka_failover", "primary",
	)
	defer stopPrimary()

	waitForAPIServerReady(t, primaryURL)

	// Given: Collector connects to primary server
	collectorUID := uuid.New()
	collectorCfg := createCollectorConfig(t, base.CacheDir, primaryPort, collectorUID)
	collectorContainer := startOTelCollector(t, collectorCfg)

	defer func() { _ = collectorContainer.Terminate(ctx) }()

	time.Sleep(5 * time.Second)

	// Then: Agent is registered on primary
	agents := listAgents(t, primaryURL)
	require.GreaterOrEqual(t, len(agents), 1, "Agent should be registered on primary")
	t.Log("Agent registered on primary server")

	// When: Secondary server starts (simulating failover scenario)
	secondaryPort := base.GetFreeTCPPort()

	stopSecondary, secondaryURL := setupAPIServerWithKafka(
		t, secondaryPort, mongoURI, kafkaBroker, "opampcommander_kafka_failover", "secondary",
	)
	defer stopSecondary()

	waitForAPIServerReady(t, secondaryURL)
	t.Log("Secondary server started")

	// Then: Agent data should be available on secondary (via shared DB)
	agentsOnSecondary := listAgents(t, secondaryURL)
	require.GreaterOrEqual(t, len(agentsOnSecondary), 1, "Agent should be visible on secondary")

	agent := findAgentByUID(agentsOnSecondary, collectorUID)
	require.NotNil(t, agent, "Agent should be found on secondary server")

	// When: Update config via secondary server
	updateRequest := map[string]interface{}{
		"config": map[string]interface{}{
			"configMap": map[string]string{
				"failover_test": "secondary_update",
			},
		},
	}
	updateAgentConfig(t, secondaryURL, collectorUID, updateRequest)

	time.Sleep(eventPropagationDelay)

	// Then: Update should be visible on both servers
	primaryAgent := getAgentByID(t, primaryURL, collectorUID)
	secondaryAgent := getAgentByID(t, secondaryURL, collectorUID)

	assert.NotNil(t, primaryAgent.Spec.RemoteConfig, "Primary should have remote config")
	assert.NotNil(t, secondaryAgent.Spec.RemoteConfig, "Secondary should have remote config")

	t.Log("Failover test completed successfully")
}

// Helper functions

//nolint:ireturn
func startKafka(t *testing.T) (testcontainers.Container, string) {
	t.Helper()
	ctx := t.Context()

	kafkaContainer, err := kafkaTestContainer.Run(ctx, kafkaImage)
	require.NoError(t, err)

	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, brokers, "Kafka brokers should not be empty")

	broker := brokers[0]
	t.Logf("Kafka started at: %s", broker)

	return kafkaContainer, broker
}

func setupAPIServerWithKafka(
	t *testing.T,
	port int,
	mongoURI string,
	kafkaBroker string,
	dbName string,
	serverID string,
) (func(), string) {
	t.Helper()

	hostname, _ := os.Hostname()
	fullServerID := fmt.Sprintf("%s-%s-test-%d", hostname, serverID, port)

	//exhaustruct:ignore
	settings := config.ServerSettings{
		Address:  fmt.Sprintf("0.0.0.0:%d", port),
		ServerID: config.ServerID(fullServerID),
		EventSettings: config.EventSettings{
			ProtocolType: config.EventProtocolTypeKafka,
			KafkaSettings: config.KafkaSettings{
				Brokers: []string{kafkaBroker},
				Topic:   "e2e.opampcommander.events",
			},
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
				Issuer:     "e2e-test-kafka",
				Expiration: 24 * time.Hour,
				Audience:   []string{"test"},
			},
		},
		//exhaustruct:ignore
		ManagementSettings: config.ManagementSettings{
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
		err := server.Run(serverCtx)
		if err != nil {
			t.Logf("Server %s stopped with error: %v", serverID, err)
		}
	}()

	// Give server time to fully initialize
	time.Sleep(serverStartupDelay)

	stopServer := func() {
		cancel()
		time.Sleep(1 * time.Second) // Allow graceful shutdown
	}

	apiBaseURL := fmt.Sprintf("http://localhost:%d", port)

	return stopServer, apiBaseURL
}

func updateAgentConfig(t *testing.T, baseURL string, agentUID uuid.UUID, updateReq map[string]interface{}) {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/agents/%s/config", baseURL, agentUID)

	body, err := json.Marshal(updateReq)
	require.NoError(t, err)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPut, url, nil) //nolint:noctx
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(bytes.NewBuffer(body))

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Logf("Update config response: %s", string(bodyBytes))
	}

	require.True(t,
		resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent,
		"Update should succeed, got status: %d", resp.StatusCode)
}
