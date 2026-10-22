//go:build e2e

package apiserver_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestE2E_Conformance_CollectorStableInstanceUID(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	testcontainers.SkipIfProviderIsNotHealthy(t)

	base := testutil.NewBase(t)
	server := base.StartStandaloneAPIServer()
	server.WaitForReady()
	apiClient := server.Client()
	uid := uuid.New()

	const image = "otel/opentelemetry-collector-contrib:0.163.2"

	collector := base.StartOTelCollectorWithIdentity(server.Port, image, uid, "before-restart")
	testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool {
		return a.Status.Connected &&
			a.Metadata.Description.IdentifyingAttributes["service.instance.id"] == "before-restart"
	}, conformanceTimeout, conformancePoll, "the configured OpAMP UID must be independent of service.instance.id")

	// Remote config is group-owned; the direct extension retains the assignment without applying it.
	configName := "pipeline"
	_, err := apiClient.AgentGroupService.CreateAgentGroup(t.Context(), "default", &v1.AgentGroup{
		Metadata: v1.Metadata{Name: "identity-config"},
		Spec: v1.Spec{
			Selector: v1.AgentSelector{IdentifyingAttributes: map[string]string{"service.name": "otelcol-contrib"}},
			AgentConfig: &v1.AgentConfig{AgentRemoteConfigs: []v1.AgentGroupRemoteConfig{{
				AgentRemoteConfigName: &configName,
				AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{Value: "receivers: {}\n", ContentType: "text/yaml"},
			}}},
		},
	})
	require.NoError(t, err)
	configured := testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool {
		return len(a.Spec.RemoteConfig.RemoteConfigNames) == 1
	}, conformanceTimeout, conformancePoll, "desired config should be assigned before restart")
	require.Equal(t, []string{"identity-config/pipeline"}, configured.Spec.RemoteConfig.RemoteConfigNames)

	stopTimeout := 10 * time.Second
	require.NoError(t, collector.Stop(t.Context(), &stopTimeout))
	// Start the replacement immediately, without waiting for Commander's close cleanup.
	base.StartOTelCollectorWithIdentity(server.Port, image, uid, "after-restart")
	resumed := testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool {
		return a.Status.Connected &&
			a.Metadata.Description.IdentifyingAttributes["service.instance.id"] == "after-restart"
	}, conformanceTimeout, conformancePoll, "restart must update the existing agent without assigning a new UID")
	assert.Equal(t, configured.Spec.RemoteConfig, resumed.Spec.RemoteConfig)

	for _, condition := range resumed.Status.Conditions {
		assert.NotEqual(t, "InstanceUIDConflict", condition.Type)
	}

	agents, err := apiClient.AgentService.ListAgents(t.Context(), "default")
	require.NoError(t, err)
	require.Len(t, agents.Items, 1, "restart must not split the agent into another record")
	assert.Equal(t, uid, agents.Items[0].Metadata.InstanceUID)
}
