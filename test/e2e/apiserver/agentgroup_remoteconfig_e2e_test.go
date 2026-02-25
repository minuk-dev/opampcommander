//go:build e2e

package apiserver_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// TestE2E_AgentGroup_RemoteConfig_DirectMode tests the remote config propagation
// when using inline/direct config definition in AgentGroup.
func TestE2E_AgentGroup_RemoteConfig_DirectMode(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()
	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_agentgroup_direct")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: Create opampctl client
	opampClient := createOpampClient(t, apiBaseURL)

	// Given: OTel Collector is started
	collectorUID := uuid.New()
	collectorCfg := createCollectorConfigWithAttributes(t, base.CacheDir, apiPort, collectorUID, map[string]string{
		"service.name": "staging-service",
		"environment":  "staging",
	})
	collectorContainer := startOTelCollector(t, collectorCfg)
	defer func() { _ = collectorContainer.Terminate(ctx) }()

	// Wait for collector to register
	assert.Eventually(t, func() bool {
		agents, err := opampClient.AgentService.ListAgents(ctx)
		if err != nil {
			return false
		}
		return len(agents.Items) > 0
	}, 2*time.Minute, 1*time.Second, "Collector should register")

	// Step 1: Create AgentGroup with inline/direct config
	agentGroupName := "staging-group"
	inlineConfigName := "collector-config"
	inlineConfigValue := `exporters:
  debug:
    verbosity: detailed
  otlp:
    endpoint: localhost:4317`

	//exhaustruct:ignore
	agentGroup, err := opampClient.AgentGroupService.CreateAgentGroup(ctx, &v1.AgentGroup{
		Metadata: v1.Metadata{
			Name: agentGroupName,
		},
		Spec: v1.Spec{
			Priority: 5,
			Selector: v1.AgentSelector{
				IdentifyingAttributes: map[string]string{
					"service.name": "staging-service",
				},
			},
			AgentConfig: &v1.AgentConfig{
				AgentRemoteConfig: &v1.AgentGroupRemoteConfig{
					AgentRemoteConfigName: &inlineConfigName,
					AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{
						Value:       inlineConfigValue,
						ContentType: "application/yaml",
					},
				},
			},
		},
	})
	require.NoError(t, err, "Failed to create AgentGroup with inline config")
	require.Equal(t, agentGroupName, agentGroup.Metadata.Name)

	t.Logf("Created AgentGroup: %s with inline config: %s", agentGroupName, inlineConfigName)

	// Step 2: Verify agents in the group receive the config with prefixed name
	// The config name should be: {AgentGroupName}/{AgentRemoteConfigName}
	expectedConfigName := fmt.Sprintf("%s/%s", agentGroupName, inlineConfigName)

	assert.Eventually(t, func() bool {
		agents, err := opampClient.AgentGroupService.ListAgentsByAgentGroup(ctx, agentGroupName)
		if err != nil {
			t.Logf("Failed to list agents by group: %v", err)
			return false
		}

		if len(agents.Items) == 0 {
			t.Log("No agents in group yet")
			return false
		}

		for _, agent := range agents.Items {
			// The config should be applied with prefixed name for inline configs
			if hasRemoteConfig(agent, expectedConfigName) {
				t.Logf("Agent %s has remote config %s applied (prefixed)", agent.Metadata.InstanceUID, expectedConfigName)
				return true
			}
			// Log current state for debugging
			t.Logf("Agent %s remote config names: %v", agent.Metadata.InstanceUID, agent.Spec.RemoteConfig.RemoteConfigNames)
			t.Logf("Agent %s effective config keys: %v", agent.Metadata.InstanceUID, getEffectiveConfigKeys(agent))
		}

		return false
	}, 2*time.Minute, 2*time.Second, "Agent should receive remote config with prefixed name")

	// Cleanup
	err = opampClient.AgentGroupService.DeleteAgentGroup(ctx, agentGroupName)
	require.NoError(t, err)
}

// TestE2E_AgentGroup_RemoteConfig_NameCollision tests that same config names
// in different AgentGroups don't collide due to prefixing.
func TestE2E_AgentGroup_RemoteConfig_NameCollision(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()
	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_agentgroup_collision")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	// Given: Create opampctl client
	opampClient := createOpampClient(t, apiBaseURL)

	// Given: Two OTel Collectors with different service names
	collectorUID1 := uuid.New()
	collectorCfg1 := createCollectorConfigWithAttributes(t, base.CacheDir, apiPort, collectorUID1, map[string]string{
		"service.name": "service-alpha",
	})
	collectorContainer1 := startOTelCollector(t, collectorCfg1)
	defer func() { _ = collectorContainer1.Terminate(ctx) }()

	collectorUID2 := uuid.New()
	collectorCfg2 := createCollectorConfigWithAttributes(t, base.CacheDir, apiPort, collectorUID2, map[string]string{
		"service.name": "service-beta",
	})
	collectorContainer2 := startOTelCollector(t, collectorCfg2)
	defer func() { _ = collectorContainer2.Terminate(ctx) }()

	// Wait for both collectors to register
	assert.Eventually(t, func() bool {
		agents, err := opampClient.AgentService.ListAgents(ctx)
		if err != nil {
			return false
		}
		return len(agents.Items) >= 2
	}, 2*time.Minute, 1*time.Second, "Both collectors should register")

	// Step 1: Create two AgentGroups with same inline config name
	commonConfigName := "config" // Same name in both groups

	// Group Alpha
	groupAlphaName := "group-alpha"
	//exhaustruct:ignore
	_, err := opampClient.AgentGroupService.CreateAgentGroup(ctx, &v1.AgentGroup{
		Metadata: v1.Metadata{
			Name: groupAlphaName,
		},
		Spec: v1.Spec{
			Priority: 10,
			Selector: v1.AgentSelector{
				IdentifyingAttributes: map[string]string{
					"service.name": "service-alpha",
				},
			},
			AgentConfig: &v1.AgentConfig{
				AgentRemoteConfig: &v1.AgentGroupRemoteConfig{
					AgentRemoteConfigName: &commonConfigName,
					AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{
						Value:       "content: alpha-specific",
						ContentType: "text/plain",
					},
				},
			},
		},
	})
	require.NoError(t, err, "Failed to create group-alpha")

	// Group Beta
	groupBetaName := "group-beta"
	//exhaustruct:ignore
	_, err = opampClient.AgentGroupService.CreateAgentGroup(ctx, &v1.AgentGroup{
		Metadata: v1.Metadata{
			Name: groupBetaName,
		},
		Spec: v1.Spec{
			Priority: 10,
			Selector: v1.AgentSelector{
				IdentifyingAttributes: map[string]string{
					"service.name": "service-beta",
				},
			},
			AgentConfig: &v1.AgentConfig{
				AgentRemoteConfig: &v1.AgentGroupRemoteConfig{
					AgentRemoteConfigName: &commonConfigName,
					AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{
						Value:       "content: beta-specific",
						ContentType: "text/plain",
					},
				},
			},
		},
	})
	require.NoError(t, err, "Failed to create group-beta")

	t.Logf("Created two groups with same config name: %s", commonConfigName)

	// Step 2: Verify each agent gets the correct prefixed config
	expectedAlphaConfig := fmt.Sprintf("%s/%s", groupAlphaName, commonConfigName)
	expectedBetaConfig := fmt.Sprintf("%s/%s", groupBetaName, commonConfigName)

	// Verify alpha agent
	assert.Eventually(t, func() bool {
		agents, err := opampClient.AgentGroupService.ListAgentsByAgentGroup(ctx, groupAlphaName)
		if err != nil || len(agents.Items) == 0 {
			return false
		}

		for _, agent := range agents.Items {
			if hasRemoteConfig(agent, expectedAlphaConfig) {
				t.Logf("Alpha agent has config: %s", expectedAlphaConfig)
				return true
			}
		}

		return false
	}, 2*time.Minute, 2*time.Second, "Alpha agent should have prefixed config")

	// Verify beta agent
	assert.Eventually(t, func() bool {
		agents, err := opampClient.AgentGroupService.ListAgentsByAgentGroup(ctx, groupBetaName)
		if err != nil || len(agents.Items) == 0 {
			return false
		}

		for _, agent := range agents.Items {
			if hasRemoteConfig(agent, expectedBetaConfig) {
				t.Logf("Beta agent has config: %s", expectedBetaConfig)
				return true
			}
		}

		return false
	}, 2*time.Minute, 2*time.Second, "Beta agent should have prefixed config")

	// Cleanup
	_ = opampClient.AgentGroupService.DeleteAgentGroup(ctx, groupAlphaName)
	_ = opampClient.AgentGroupService.DeleteAgentGroup(ctx, groupBetaName)
}

// Helper functions

func createOpampClient(t *testing.T, baseURL string) *client.Client {
	t.Helper()

	opampClient := client.New(baseURL)

	// Get auth token
	token := getAuthToken(t, baseURL)
	opampClient.SetAuthToken(token)

	return opampClient
}

func hasRemoteConfig(agent v1.Agent, configName string) bool {
	// Check RemoteConfigNames list
	for _, name := range agent.Spec.RemoteConfig.RemoteConfigNames {
		if name == configName {
			return true
		}
	}

	// Also check effective config's ConfigMap as fallback
	if agent.Status.EffectiveConfig.ConfigMap.ConfigMap != nil {
		if _, exists := agent.Status.EffectiveConfig.ConfigMap.ConfigMap[configName]; exists {
			return true
		}
	}

	return false
}

func createCollectorConfigWithAttributes(
	t *testing.T,
	cacheDir string,
	opampPort int,
	instanceUID uuid.UUID,
	resourceAttrs map[string]string,
) string {
	t.Helper()

	// Build resource attributes string for OTEL_RESOURCE_ATTRIBUTES
	var resourceAttrsStr string
	for k, v := range resourceAttrs {
		if resourceAttrsStr != "" {
			resourceAttrsStr += ","
		}
		resourceAttrsStr += fmt.Sprintf("%s=%s", k, v)
	}

	configContent := fmt.Sprintf(`receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
  resource:
    attributes:
%s

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
    instance_uid: %s

service:
  extensions: [opamp]
  telemetry:
    logs:
      level: info
    resource:
%s
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch, resource]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      processors: [batch, resource]
      exporters: [debug]
    logs:
      receivers: [otlp]
      processors: [batch, resource]
      exporters: [debug]
`, formatResourceAttrsForConfig(resourceAttrs), opampPort, instanceUID.String(), formatResourceAttrsForTelemetry(resourceAttrs))

	configPath := filepath.Join(cacheDir, fmt.Sprintf("collector-config-%s.yaml", instanceUID.String()))
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	return configPath
}

func formatResourceAttrsForConfig(attrs map[string]string) string {
	var result string
	for k, v := range attrs {
		result += fmt.Sprintf("      - key: %s\n        value: %s\n        action: upsert\n", k, v)
	}
	return result
}

func formatResourceAttrsForTelemetry(attrs map[string]string) string {
	var result string
	for k, v := range attrs {
		result += fmt.Sprintf("      %s: %s\n", k, v)
	}
	return result
}

func getEffectiveConfigKeys(agent v1.Agent) []string {
	keys := make([]string, 0)
	if agent.Status.EffectiveConfig.ConfigMap.ConfigMap != nil {
		for k := range agent.Status.EffectiveConfig.ConfigMap.ConfigMap {
			keys = append(keys, k)
		}
	}
	return keys
}
