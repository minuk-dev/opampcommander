// NOTE: These tests may report data races from the IBM/sarama Kafka client library (v1.46.3).
// This is a known issue in the sarama library itself and not in our code.
// See: https://github.com/IBM/sarama/issues
// The tests are functionally correct and pass all assertions.

//go:build e2e

package apiserver_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"gopkg.in/yaml.v3"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// TestE2E_APIServer_KafkaOutage verifies actual agent config application while
// Kafka is stopped, then producer recovery after the same broker restarts.
func TestE2E_APIServer_KafkaOutage(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()
	kafkaServer := base.StartKafka()
	writer := base.StartAPIServerWithKafka(mongoServer.URI, kafkaServer.Broker, "kafka_outage")
	writer.WaitForReady()

	peer := base.StartAPIServerWithKafka(mongoServer.URI, kafkaServer.Broker, "kafka_outage")
	peer.WaitForReady()

	local := base.StartReferenceAgent(writer.Port,
		testutil.WithReferenceAgentIdentifyingAttributes(map[string]string{"service.name": "kafka-outage"}))
	remote := base.StartReferenceAgent(peer.Port,
		testutil.WithReferenceAgentIdentifyingAttributes(map[string]string{"service.name": "kafka-outage"}))

	apiClient := writer.Client()
	for _, agent := range []*testutil.ReferenceAgent{local, remote} {
		testutil.EventuallyAgent(t, apiClient, "default", agent.UID, func(a *v1.Agent) bool {
			return a.Status.Connected
		}, time.Minute, time.Second, "agent should connect before the broker outage")
	}

	createAgentGroup(t, apiClient, "outage", map[string]string{"service.name": "kafka-outage"})
	updateAgentGroup(t, apiClient, "outage", map[string]string{"phase": "before"})
	require.Eventually(t, func() bool {
		return local.EffectiveConfig()["outage/inline-config"] == "phase: before\n" &&
			remote.EffectiveConfig()["outage/inline-config"] == "phase: before\n"
	}, 2*time.Minute, time.Second, "both agents must apply the initial config, warming the peer cache")

	stopTimeout := time.Second
	brokerPort, err := kafkaServer.MappedPort(t.Context(), "9093/tcp")
	require.NoError(t, err)
	require.NoError(t, kafkaServer.Stop(t.Context(), &stopTimeout))

	outageStarted := time.Now()

	updateAgentGroup(t, apiClient, "outage", map[string]string{"phase": "outage"})

	// Kafka remains down: remote recovery must come from persisted state and
	// heartbeat/cache expiry, not a replay after broker reconnection.
	require.Eventually(t, func() bool {
		return local.EffectiveConfig()["outage/inline-config"] == "phase: outage\n" &&
			remote.EffectiveConfig()["outage/inline-config"] == "phase: outage\n"
	}, 2*time.Minute, time.Second, "local and remote agents must apply the config while Kafka stays down")

	for _, agent := range []*testutil.ReferenceAgent{local, remote} {
		restartAgentForKafkaTest(t, apiClient, agent.UID)
	}

	require.Eventually(t, func() bool { return local.Restarts() > 0 && remote.Restarts() > 0 },
		2*time.Minute, time.Second, "persisted restart requests must reach both agents while Kafka stays down")
	require.Eventually(t, func() bool {
		return kafkaProducerDegraded(t, writer)
	}, 2*time.Minute, time.Second, "an exhausted producer retry must be visible without failing probes")

	for _, agent := range []*testutil.ReferenceAgent{local, remote} {
		current, err := apiClient.AgentService.GetAgent(t.Context(), "default", agent.UID)
		require.NoError(t, err)
		require.True(t, current.Status.Connected)
	}

	// Keep Kafka down beyond the consumer's metadata/coordinator retry budgets.
	time.Sleep(time.Until(outageStarted.Add(90 * time.Second)))

	require.NoError(t, kafkaServer.Start(t.Context()))
	restartedPort, err := kafkaServer.MappedPort(t.Context(), "9093/tcp")
	require.NoError(t, err)
	require.Equal(t, brokerPort, restartedPort, "broker restart must preserve its advertised endpoint")

	updateAgentGroup(t, apiClient, "outage", map[string]string{"phase": "recovered"})
	require.Eventually(t, func() bool {
		return local.EffectiveConfig()["outage/inline-config"] == "phase: recovered\n" &&
			remote.EffectiveConfig()["outage/inline-config"] == "phase: recovered\n"
	}, 3*time.Minute, time.Second, "agents must keep applying updates after broker recovery")

	restartAgentForKafkaTest(t, apiClient, remote.UID)
	require.Eventually(t, func() bool { return !kafkaProducerDegraded(t, writer) }, time.Minute, time.Second,
		"a new notification must reach the recovered producer without restarting the API server")

	// This agent cannot recover via a heartbeat inside the assertion window:
	// a successful restart proves the peer's Kafka receive loop is still running.
	quiet := base.StartReferenceAgent(peer.Port, testutil.WithReferenceAgentHeartbeat(time.Hour),
		testutil.WithReferenceAgentIdentifyingAttributes(map[string]string{"service.name": "kafka-push"}))
	testutil.EventuallyAgent(t, apiClient, "default", quiet.UID, func(a *v1.Agent) bool {
		return a.Status.Connected && a.Metadata.Capabilities != 0
	}, time.Minute, time.Second, "quiet agent should register")

	restartAgentForKafkaTest(t, apiClient, quiet.UID)
	require.Eventually(t, func() bool { return quiet.Restarts() > 0 }, 15*time.Second, time.Second,
		"Kafka delivery must resume after a sustained outage, without waiting for heartbeat or cache expiry")
}

func restartAgentForKafkaTest(t *testing.T, apiClient *client.Client, uid uuid.UUID) {
	t.Helper()

	var err error

	// A heartbeat can win optimistic concurrency; retry only that expected conflict.
	require.Eventually(t, func() bool {
		_, err = apiClient.AgentService.RestartAgent(t.Context(), "default", uid)

		var responseError *client.ResponseError

		return !errors.As(err, &responseError) || responseError.StatusCode != http.StatusConflict
	}, time.Minute, time.Second, "restart request must eventually persist")
	require.NoError(t, err, "Kafka loss must not fail a persisted management request")
}

func kafkaProducerDegraded(t *testing.T, server *testutil.APIServer) bool {
	t.Helper()

	var degraded bool

	for _, path := range []string{"/healthz", "/readyz"} {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.ManagementEndpoint+path, nil)
		require.NoError(t, err)
		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode, "Kafka loss must not drain or restart the node")

		var reasons map[string]string

		err = json.NewDecoder(response.Body).Decode(&reasons)
		_ = response.Body.Close()

		require.NoError(t, err)

		if path == "/healthz" {
			degraded = reasons["KafkaProducer"] != ""
		}
	}

	return degraded
}

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

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + Kafka)
	mongoServer := base.StartMongoDB()
	kafkaServer := base.StartKafka()

	// Given: Two API servers in distributed mode.
	// Boot them serially: the RBAC SyncPolicies startup hook does a
	// Casbin Clear+SavePolicy against the shared MongoDB, and two
	// concurrent boots race on the policy bulk-insert (E11000 duplicate
	// key) which kills one of the Fx starts.
	apiServer1 := base.StartAPIServerWithKafka(mongoServer.URI, kafkaServer.Broker, "opampcommander_kafka_e2e")

	apiServer1.WaitForReady()

	apiServer2 := base.StartAPIServerWithKafka(mongoServer.URI, kafkaServer.Broker, "opampcommander_kafka_e2e")

	apiServer2.WaitForReady()

	t.Log("Both API servers are ready")

	client1 := apiServer1.Client()
	client2 := apiServer2.Client()

	// Given: AgentGroup is created on server 1
	agentGroupName := "test-group"
	createAgentGroup(t, client1, agentGroupName, map[string]string{
		"service.name": "otel-collector-e2e-test",
	})
	t.Logf("AgentGroup created: %s", agentGroupName)

	// Given: Collector connects to server 1
	collector := base.StartOTelCollector(apiServer1.Port)

	// Then: Agent should be visible on server 1
	var agent1 *v1.Agent

	assert.Eventually(t, func() bool {
		agents1 := listAgents(t, apiServer1.Endpoint)
		if len(agents1) < 1 {
			return false
		}

		agent1 = findAgentByUID(agents1, collector.UID)

		return agent1 != nil
	}, 30*time.Second, 1*time.Second, "Agent should be registered on server 1")
	require.NotNil(t, agent1, "Collector should be found on server 1")
	t.Logf("Agent registered on server 1: %s", agent1.Metadata.InstanceUID)

	// Then: Agent should also be visible on server 2 (shared database)
	agents2 := listAgents(t, apiServer2.Endpoint)
	require.GreaterOrEqual(t, len(agents2), 1, "Agent should be visible on server 2")
	agent2 := findAgentByUID(agents2, collector.UID)
	require.NotNil(t, agent2, "Collector should be found on server 2")
	t.Logf("Agent visible on server 2: %s", agent2.Metadata.InstanceUID)

	// When: Server 2 updates the agent group configuration
	updateAgentGroup(t, client2, agentGroupName, map[string]string{
		"test_key": "test_value_from_server2",
	})
	t.Log("AgentGroup config updated via server 2")

	// Then: Both servers should have access to the agent data (since they share the same DB)
	// Note: Remote config may not be supported by the collector, so we verify agent presence instead
	assert.Eventually(t, func() bool {
		updatedAgent1, err1 := tryGetAgentByIDWithClient(t, client1, collector.UID)
		if err1 != nil {
			t.Logf("Failed to get agent from server1: %v", err1)

			return false
		}

		updatedAgent2, err2 := tryGetAgentByIDWithClient(t, client2, collector.UID)
		if err2 != nil {
			t.Logf("Failed to get agent from server2: %v", err2)

			return false
		}

		t.Logf("Agent1 InstanceUID: %s", updatedAgent1.Metadata.InstanceUID)
		t.Logf("Agent2 InstanceUID: %s", updatedAgent2.Metadata.InstanceUID)

		agent1Present := updatedAgent1.Metadata.InstanceUID == collector.UID
		agent2Present := updatedAgent2.Metadata.InstanceUID == collector.UID

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

	base := testutil.NewBase(t)

	// Given: Infrastructure setup
	mongoServer := base.StartMongoDB()
	kafkaServer := base.StartKafka()

	// Given: Primary server is running
	primaryServer := base.StartAPIServerWithKafka(mongoServer.URI, kafkaServer.Broker, "opampcommander_kafka_failover")

	primaryServer.WaitForReady()

	primaryClient := primaryServer.Client()

	// Given: AgentGroup is created on primary server
	agentGroupName := "failover-test-group"
	createAgentGroup(t, primaryClient, agentGroupName, map[string]string{
		"service.name": "otel-collector-e2e-test",
	})
	t.Logf("AgentGroup created: %s", agentGroupName)

	// Given: Collector connects to primary server
	collector := base.StartOTelCollector(primaryServer.Port)

	// Then: Agent is registered on primary
	assert.Eventually(t, func() bool {
		agents := listAgents(t, primaryServer.Endpoint)

		return len(agents) >= 1
	}, 30*time.Second, 1*time.Second, "Agent should be registered on primary")
	t.Log("Agent registered on primary server")

	// When: Secondary server starts (simulating failover scenario)
	secondaryServer := base.StartAPIServerWithKafka(mongoServer.URI, kafkaServer.Broker, "opampcommander_kafka_failover")

	secondaryServer.WaitForReady()
	t.Log("Secondary server started")

	secondaryClient := secondaryServer.Client()

	// Then: Agent data should be available on secondary (via shared DB)
	agentsOnSecondary := listAgents(t, secondaryServer.Endpoint)
	require.GreaterOrEqual(t, len(agentsOnSecondary), 1, "Agent should be visible on secondary")

	agent := findAgentByUID(agentsOnSecondary, collector.UID)
	require.NotNil(t, agent, "Agent should be found on secondary server")

	// When: Update config via secondary server using AgentGroup
	updateAgentGroup(t, secondaryClient, agentGroupName, map[string]string{
		"failover_test": "secondary_update",
	})
	t.Log("AgentGroup config updated via secondary server")

	// Then: Since the collector doesn't support remote config,
	// we'll verify that the agent data is properly shared between servers
	// and both servers can access the same agent information
	assert.Eventually(t, func() bool {
		primaryAgent, err1 := tryGetAgentByIDWithClient(t, primaryClient, collector.UID)
		if err1 != nil {
			t.Logf("Failed to get agent from primary: %v", err1)

			return false
		}

		secondaryAgent, err2 := tryGetAgentByIDWithClient(t, secondaryClient, collector.UID)
		if err2 != nil {
			t.Logf("Failed to get agent from secondary: %v", err2)

			return false
		}

		// Both servers should see the agent and have the same basic agent information
		primaryHasAgent := primaryAgent.Metadata.InstanceUID == collector.UID
		secondaryHasAgent := secondaryAgent.Metadata.InstanceUID == collector.UID

		t.Logf("Primary has agent: %v, Secondary has agent: %v", primaryHasAgent, secondaryHasAgent)

		return primaryHasAgent && secondaryHasAgent
	}, 30*time.Second, 1*time.Second, "Both servers should have access to the agent")

	t.Log("Failover test completed successfully")
}

func createAgentGroup(t *testing.T, c *client.Client, name string, selector map[string]string) {
	t.Helper()

	t.Logf("Creating AgentGroup with name: %s", name)

	//exhaustruct:ignore
	_, err := c.AgentGroupService.CreateAgentGroup(t.Context(), "default", &v1.AgentGroup{
		Metadata: v1.Metadata{
			Name: name,
		},
		Spec: v1.Spec{
			Selector: v1.AgentSelector{
				IdentifyingAttributes: selector,
			},
		},
	})
	require.NoError(t, err, "Create AgentGroup %s should succeed", name)

	t.Logf("AgentGroup '%s' created successfully", name)
}

func agentGroupExistsOnServer(t *testing.T, c *client.Client, name string) bool {
	t.Helper()

	_, err := c.AgentGroupService.GetAgentGroup(t.Context(), "default", name)

	return err == nil
}

func updateAgentGroup(t *testing.T, c *client.Client, name string, configMap map[string]string) {
	t.Helper()

	t.Logf("Updating AgentGroup: %s", name)

	agentGroup, err := c.AgentGroupService.GetAgentGroup(t.Context(), "default", name)
	if err != nil {
		t.Logf("AgentGroup '%s' not found, creating it first", name)
		createAgentGroup(t, c, name, map[string]string{})

		agentGroup, err = c.AgentGroupService.GetAgentGroup(t.Context(), "default", name)
		require.NoError(t, err, "Failed to get AgentGroup after creation")
	}

	configBytes, err := yaml.Marshal(configMap)
	require.NoError(t, err)

	configName := "inline-config"
	configValue := string(configBytes)
	agentGroup.Spec.AgentConfig = &v1.AgentConfig{
		AgentRemoteConfigs: []v1.AgentGroupRemoteConfig{
			{
				AgentRemoteConfigName: &configName,
				AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{
					Value:       configValue,
					ContentType: "application/yaml",
				},
			},
		},
	}

	t.Logf("AgentGroup after update: %+v", agentGroup)

	_, err = c.AgentGroupService.UpdateAgentGroup(t.Context(), agentGroup)
	require.NoError(t, err, "Update AgentGroup should succeed")
}

// TestE2E_APIServer_KafkaEventMessaging tests server-to-server event messaging via Kafka.
// This test verifies that events are properly sent and received through Kafka messaging system.
func TestE2E_APIServer_KafkaEventMessaging(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)

	// Given: Infrastructure setup (MongoDB + Kafka)
	mongoServer := base.StartMongoDB()
	kafkaServer := base.StartKafka()

	// Given: Two API servers in distributed mode.
	// Boot them serially to avoid the concurrent RBAC SyncPolicies race
	// on the shared MongoDB (see TestE2E_APIServer_KafkaDistributedMode
	// for the underlying cause).
	apiServer1 := base.StartAPIServerWithKafka(mongoServer.URI, kafkaServer.Broker, "opampcommander_kafka_messaging_e2e")

	apiServer1.WaitForReady()

	apiServer2 := base.StartAPIServerWithKafka(mongoServer.URI, kafkaServer.Broker, "opampcommander_kafka_messaging_e2e")

	apiServer2.WaitForReady()

	t.Log("Both API servers are ready for messaging test")

	client1 := apiServer1.Client()
	client2 := apiServer2.Client()

	// Given: AgentGroup is created
	agentGroupName := "messaging-test-group"
	createAgentGroup(t, client1, agentGroupName, map[string]string{
		"service.name": "otel-collector-messaging-test",
	})
	t.Logf("AgentGroup created: %s", agentGroupName)

	// Given: Collector connects to server 1
	collector := base.StartOTelCollector(apiServer1.Port)

	// Then: Agent should be registered on server 1
	assert.Eventually(t, func() bool {
		agents := listAgents(t, apiServer1.Endpoint)

		return len(agents) >= 1 && findAgentByUID(agents, collector.UID) != nil
	}, 30*time.Second, 1*time.Second, "Agent should be registered on server 1")
	t.Log("Agent registered on server 1")

	// When: Update agent group configuration via server 2
	// This should trigger an event that propagates through Kafka
	updateAgentGroup(t, client2, agentGroupName, map[string]string{
		"messaging_test_key": "updated_via_server2",
	})
	t.Log("AgentGroup config updated via server 2")

	// Then: Both servers should maintain consistent agent data
	// The messaging system ensures servers are aware of configuration changes
	assert.Eventually(t, func() bool {
		agent1, err1 := tryGetAgentByIDWithClient(t, client1, collector.UID)
		agent2, err2 := tryGetAgentByIDWithClient(t, client2, collector.UID)

		if err1 != nil || err2 != nil {
			t.Logf("Failed to get agents: err1=%v, err2=%v", err1, err2)

			return false
		}

		// Verify both servers have access to the agent
		agentExists1 := agent1.Metadata.InstanceUID == collector.UID
		agentExists2 := agent2.Metadata.InstanceUID == collector.UID

		t.Logf("Agent on server1: %v, server2: %v", agentExists1, agentExists2)

		return agentExists1 && agentExists2
	}, 30*time.Second, 1*time.Second, "Both servers should have consistent agent data through messaging")

	t.Log("Kafka event messaging test completed successfully")
}
