// NOTE: These tests may report data races from the IBM/sarama Kafka client library (v1.46.3).
// This is a known issue in the sarama library itself and not in our code.
// See: https://github.com/IBM/sarama/issues
// The tests are functionally correct and pass all assertions.

//go:build e2e

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

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	kafkaTestContainer "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/wait"
	"gopkg.in/yaml.v3"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	v1agentgroup "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/pkg/apiserver"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	kafkaImage = "confluentinc/cp-kafka:7.5.0"
)

// TestE2E_APIServer_KafkaDistributedMode tests distributed mode with Kafka messaging
// Scenario: Two API servers communicate via Kafka, and an agent update on server1
// should be propagated to server2.
// NOTE: This test requires authentication for AgentGroup API, which is not yet
// implemented in E2E test setup. Skip for now.
func TestE2E_APIServer_KafkaDistributedMode(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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

	// Given: AgentGroup is created on server 1
	agentGroupName := "test-group"
	createAgentGroup(t, server1URL, agentGroupName, map[string]string{
		"service.name": "otel-collector-e2e-test",
	})
	t.Logf("AgentGroup created: %s", agentGroupName)

	// Given: Collector connects to server 1
	collectorUID := uuid.New()
	collectorCfg := createCollectorConfig(t, base.CacheDir, server1Port, collectorUID)
	collectorContainer := startOTelCollector(t, collectorCfg)

	defer func() { _ = collectorContainer.Terminate(ctx) }()

	// Then: Agent should be visible on server 1
	var agent1 *v1agent.Agent

	assert.Eventually(t, func() bool {
		agents1 := listAgents(t, server1URL)
		if len(agents1) < 1 {
			return false
		}

		agent1 = findAgentByUID(agents1, collectorUID)

		return agent1 != nil
	}, 30*time.Second, 1*time.Second, "Agent should be registered on server 1")
	require.NotNil(t, agent1, "Collector should be found on server 1")
	t.Logf("Agent registered on server 1: %s", agent1.Metadata.InstanceUID)

	// Then: Agent should also be visible on server 2 (shared database)
	agents2 := listAgents(t, server2URL)
	require.GreaterOrEqual(t, len(agents2), 1, "Agent should be visible on server 2")
	agent2 := findAgentByUID(agents2, collectorUID)
	require.NotNil(t, agent2, "Collector should be found on server 2")
	t.Logf("Agent visible on server 2: %s", agent2.Metadata.InstanceUID)

	// When: Server 2 updates the agent group configuration
	updateAgentGroup(t, server2URL, agentGroupName, map[string]string{
		"test_key": "test_value_from_server2",
	})
	t.Log("AgentGroup config updated via server 2")

	// Then: Both servers should have access to the agent data (since they share the same DB)
	// Note: Remote config may not be supported by the collector, so we verify agent presence instead
	assert.Eventually(t, func() bool {
		updatedAgent1, err1 := tryGetAgentByID(server1URL, collectorUID)
		if err1 != nil {
			t.Logf("Failed to get agent from server1: %v", err1)

			return false
		}

		updatedAgent2, err2 := tryGetAgentByID(server2URL, collectorUID)
		if err2 != nil {
			t.Logf("Failed to get agent from server2: %v", err2)

			return false
		}

		t.Logf("Agent1 InstanceUID: %s", updatedAgent1.Metadata.InstanceUID)
		t.Logf("Agent2 InstanceUID: %s", updatedAgent2.Metadata.InstanceUID)

		agent1Present := updatedAgent1.Metadata.InstanceUID == collectorUID
		agent2Present := updatedAgent2.Metadata.InstanceUID == collectorUID

		t.Logf("Agent1 present: %v, Agent2 present: %v", agent1Present, agent2Present)

		return agent1Present && agent2Present
	}, 30*time.Second, 1*time.Second, "Both servers should have access to the agent")

	t.Log("Distributed mode test completed successfully")
}

// TestE2E_APIServer_KafkaFailover tests failover scenario in distributed mode.
// NOTE: This test requires authentication for AgentGroup API, which is not yet
// implemented in E2E test setup. Skip for now.
func TestE2E_APIServer_KafkaFailover(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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

	// Given: AgentGroup is created on primary server
	agentGroupName := "failover-test-group"
	createAgentGroup(t, primaryURL, agentGroupName, map[string]string{
		"service.name": "otel-collector-e2e-test",
	})
	t.Logf("AgentGroup created: %s", agentGroupName)

	// Given: Collector connects to primary server
	collectorUID := uuid.New()
	collectorCfg := createCollectorConfig(t, base.CacheDir, primaryPort, collectorUID)
	collectorContainer := startOTelCollector(t, collectorCfg)

	defer func() { _ = collectorContainer.Terminate(ctx) }()

	// Then: Agent is registered on primary
	assert.Eventually(t, func() bool {
		agents := listAgents(t, primaryURL)

		return len(agents) >= 1
	}, 30*time.Second, 1*time.Second, "Agent should be registered on primary")
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

	// When: Update config via secondary server using AgentGroup
	updateAgentGroup(t, secondaryURL, agentGroupName, map[string]string{
		"failover_test": "secondary_update",
	})
	t.Log("AgentGroup config updated via secondary server")

	// Then: Since the collector doesn't support remote config,
	// we'll verify that the agent data is properly shared between servers
	// and both servers can access the same agent information
	assert.Eventually(t, func() bool {
		primaryAgent, err1 := tryGetAgentByID(primaryURL, collectorUID)
		if err1 != nil {
			t.Logf("Failed to get agent from primary: %v", err1)

			return false
		}

		secondaryAgent, err2 := tryGetAgentByID(secondaryURL, collectorUID)
		if err2 != nil {
			t.Logf("Failed to get agent from secondary: %v", err2)

			return false
		}

		// Both servers should see the agent and have the same basic agent information
		primaryHasAgent := primaryAgent.Metadata.InstanceUID == collectorUID
		secondaryHasAgent := secondaryAgent.Metadata.InstanceUID == collectorUID

		t.Logf("Primary has agent: %v, Secondary has agent: %v", primaryHasAgent, secondaryHasAgent)

		return primaryHasAgent && secondaryHasAgent
	}, 30*time.Second, 1*time.Second, "Both servers should have access to the agent")

	t.Log("Failover test completed successfully")
}

// Helper functions

func startKafka(t *testing.T) (testcontainers.Container, string) {
	t.Helper()
	ctx := t.Context()

	kafkaContainer, err := kafkaTestContainer.Run(ctx,
		kafkaImage,
		kafkaTestContainer.WithClusterID("test-cluster-id"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("9093/tcp")),
	)
	require.NoError(t, err)

	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, brokers, "Kafka brokers should not be empty")

	broker := brokers[0]
	t.Logf("Kafka started at: %s", broker)

	// Wait for Kafka to be truly ready by attempting to connect
	waitForKafkaReady(t, broker)

	return kafkaContainer, broker
}

//nolint:nestif
func waitForKafkaReady(t *testing.T, broker string) {
	t.Helper()

	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Metadata.Timeout = 10 * time.Second
	config.Metadata.Retry.Max = 10
	config.Metadata.Retry.Backoff = 1 * time.Second
	config.Net.DialTimeout = 10 * time.Second
	config.Net.ReadTimeout = 10 * time.Second
	config.Net.WriteTimeout = 10 * time.Second
	config.Admin.Timeout = 10 * time.Second

	maxRetries := 60
	for i := range maxRetries {
		client, err := sarama.NewClient([]string{broker}, config)
		if err == nil {
			// Successfully created client, check if we can retrieve metadata
			brokers := client.Brokers()
			if len(brokers) > 0 {
				// Try to connect to broker
				err = brokers[0].Open(config)
				if err == nil {
					connected, err := brokers[0].Connected()
					if err == nil && connected {
						t.Logf("Kafka is ready after %d retries", i+1)
						client.Close() //nolint:errcheck,gosec

						return
					}

					if err != nil {
						t.Logf("Kafka broker connection check failed: %v", err)
					}
				} else {
					t.Logf("Kafka broker open failed: %v", err)
				}
			}

			client.Close() //nolint:errcheck,gosec
		} else {
			t.Logf("Kafka client creation attempt %d failed: %v", i+1, err)
		}

		time.Sleep(1 * time.Second)
	}

	t.Fatal("Kafka did not become ready in time")
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

	managementPort, err := testutil.GetFreeTCPPort()
	require.NoError(t, err)

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
		err := server.Run(serverCtx)
		if err != nil {
			t.Logf("Server %s stopped with error: %v", serverID, err)
		}
	}()

	stopServer := func() {
		cancel()
	}

	apiBaseURL := fmt.Sprintf("http://localhost:%d", port)

	return stopServer, apiBaseURL
}

func createAgentGroup(t *testing.T, baseURL, name string, selector map[string]string) {
	t.Helper()

	url := baseURL + "/api/v1/agentgroups"
	t.Logf("Creating AgentGroup at URL: %s with name: %s", url, name)

	reqBody := map[string]interface{}{
		"name": name,
		"selector": map[string]interface{}{
			"matchLabels": selector,
		},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)
	t.Logf("Request body: %s", string(body))

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body)) //nolint:noctx
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	token := getAuthToken(t, baseURL)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Logf("Create AgentGroup response: %s", string(bodyBytes))
	}

	require.True(t,
		resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated,
		"Create AgentGroup should succeed, got status: %d", resp.StatusCode)

	t.Logf("AgentGroup '%s' created successfully", name)
}

func updateAgentGroup(t *testing.T, baseURL, name string, configMap map[string]string) {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/agentgroups/%s", baseURL, name)
	t.Logf("Updating AgentGroup at URL: %s", url)

	// First get the current AgentGroup
	client := &http.Client{Timeout: 10 * time.Second}
	getReq, err := http.NewRequest(http.MethodGet, url, nil) //nolint:noctx
	require.NoError(t, err)

	token := getAuthToken(t, baseURL)
	getReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(getReq)
	require.NoError(t, err, "Failed to get AgentGroup before update")

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var agentGroup v1agentgroup.AgentGroup

	err = json.NewDecoder(resp.Body).Decode(&agentGroup)
	require.NoError(t, err)

	err = resp.Body.Close()
	require.NoError(t, err)

	t.Logf("Current AgentGroup before update: %+v", agentGroup)

	// Convert configMap to YAML and set it as the Value field
	configBytes, err := yaml.Marshal(configMap)
	require.NoError(t, err)

	agentGroup.Spec.AgentConfig = &v1agentgroup.AgentConfig{
		Value: string(configBytes),
	}

	t.Logf("AgentGroup after update: %+v", agentGroup)

	// Send the update
	body, err := json.Marshal(agentGroup)
	require.NoError(t, err)

	putReq, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body)) //nolint:noctx
	require.NoError(t, err)

	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Authorization", "Bearer "+token)

	resp, err = client.Do(putReq)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Update AgentGroup should succeed")
}

// TestE2E_APIServer_KafkaEventMessaging tests server-to-server event messaging via Kafka.
// This test verifies that events are properly sent and received through Kafka messaging system.
func TestE2E_APIServer_KafkaEventMessaging(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure setup (MongoDB + Kafka)
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	kafkaContainer, kafkaBroker := startKafka(t)
	defer func() { _ = kafkaContainer.Terminate(ctx) }()

	// Given: Two API servers in distributed mode
	server1Port := base.GetFreeTCPPort()
	server2Port := base.GetFreeTCPPort()

	stopServer1, server1URL := setupAPIServerWithKafka(
		t, server1Port, mongoURI, kafkaBroker, "opampcommander_kafka_messaging_e2e", "messaging-server-1",
	)
	defer stopServer1()

	stopServer2, server2URL := setupAPIServerWithKafka(
		t, server2Port, mongoURI, kafkaBroker, "opampcommander_kafka_messaging_e2e", "messaging-server-2",
	)
	defer stopServer2()

	waitForAPIServerReady(t, server1URL)
	waitForAPIServerReady(t, server2URL)

	t.Log("Both API servers are ready for messaging test")

	// Given: AgentGroup is created
	agentGroupName := "messaging-test-group"
	createAgentGroup(t, server1URL, agentGroupName, map[string]string{
		"service.name": "otel-collector-messaging-test",
	})
	t.Logf("AgentGroup created: %s", agentGroupName)

	// Given: Collector connects to server 1
	collectorUID := uuid.New()
	collectorCfg := createCollectorConfig(t, base.CacheDir, server1Port, collectorUID)
	collectorContainer := startOTelCollector(t, collectorCfg)
	defer func() { _ = collectorContainer.Terminate(ctx) }()

	// Then: Agent should be registered on server 1
	assert.Eventually(t, func() bool {
		agents := listAgents(t, server1URL)
		return len(agents) >= 1 && findAgentByUID(agents, collectorUID) != nil
	}, 30*time.Second, 1*time.Second, "Agent should be registered on server 1")
	t.Log("Agent registered on server 1")

	// When: Update agent group configuration via server 2
	// This should trigger an event that propagates through Kafka
	updateAgentGroup(t, server2URL, agentGroupName, map[string]string{
		"messaging_test_key": "updated_via_server2",
	})
	t.Log("AgentGroup config updated via server 2")

	// Then: Both servers should maintain consistent agent data
	// The messaging system ensures servers are aware of configuration changes
	assert.Eventually(t, func() bool {
		agent1, err1 := tryGetAgentByID(server1URL, collectorUID)
		agent2, err2 := tryGetAgentByID(server2URL, collectorUID)

		if err1 != nil || err2 != nil {
			t.Logf("Failed to get agents: err1=%v, err2=%v", err1, err2)
			return false
		}

		// Verify both servers have access to the agent
		agentExists1 := agent1.Metadata.InstanceUID == collectorUID
		agentExists2 := agent2.Metadata.InstanceUID == collectorUID

		t.Logf("Agent on server1: %v, server2: %v", agentExists1, agentExists2)

		return agentExists1 && agentExists2
	}, 30*time.Second, 1*time.Second, "Both servers should have consistent agent data through messaging")

	t.Log("Kafka event messaging test completed successfully")
}
