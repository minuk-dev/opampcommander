//go:build e2e

package apiserver_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const currentCollectorImage = "ghcr.io/open-telemetry/opentelemetry-collector-releases/" +
	"opentelemetry-collector-contrib:0.163.1"
const currentClientTimeout = 120 * time.Second

// These tests intentionally use released clients, not the reference-client fixture.
func TestE2E_CurrentCollector(t *testing.T) {
	requireCurrentClients(t)
	t.Parallel()

	for _, transport := range []string{"ws", "http"} {
		t.Run(transport, func(t *testing.T) {
			t.Parallel()
			base := testutil.NewBase(t)
			server := base.StartStandaloneAPIServer()
			server.WaitForReady()
			apiClient := server.Client()
			uid := uuid.New()
			first := startCurrentCollector(t, server.Port, uid, transport, "first")
			registered := testutil.EventuallyAgent(t, apiClient, "compatibility", uid, func(a *v1.Agent) bool {
				return a.Metadata.Description.IdentifyingAttributes["service.instance.id"] == "first" &&
					a.Status.EffectiveConfig.ConfigMap.ConfigMap["raw"].Body != ""
			}, currentClientTimeout, conformancePoll, "resource namespace and raw config should be reported")
			assert.Equal(t, "compatibility", registered.Metadata.Description.NonIdentifyingAttributes["service.namespace"])
			assert.Zero(t, uint64(registered.Metadata.Capabilities)&
				uint64(protobufs.AgentCapabilities_AgentCapabilities_AcceptsRemoteConfig))
			assert.Contains(t, registered.Status.EffectiveConfig.ConfigMap.ConfigMap["raw"].Body, "${env:COMPAT_ENDPOINT}")
			assert.Contains(t, registered.Status.EffectiveConfig.ConfigMap.ConfigMap[""].Body, "http://backend.invalid:4318")
			assertCurrentEndpoints(t, server, uid)
			require.NoError(t, first.Terminate(t.Context()))

			if transport == "ws" {
				testutil.EventuallyAgent(t, apiClient, "compatibility", uid, func(a *v1.Agent) bool {
					return !a.Status.Connected
				}, currentClientTimeout, conformancePoll, "old WebSocket should close before changed identity reconnects")
			}

			startCurrentCollector(t, server.Port, uid, transport, "second")
			testutil.EventuallyAgent(t, apiClient, "compatibility", uid, func(a *v1.Agent) bool {
				return a.Status.Connected && a.Metadata.Description.IdentifyingAttributes["service.instance.id"] == "second"
			}, currentClientTimeout, conformancePoll, "a restarted Collector should retain its OpAMP UID")
			agents, err := apiClient.AgentService.ListAgents(t.Context(), "compatibility")
			require.NoError(t, err)
			require.Len(t, agents.Items, 1, "restart must not create a second agent")
		})
	}
}

func startCurrentCollector(
	t *testing.T, port int, uid uuid.UUID, transport, serviceID string,
) testcontainers.Container {
	t.Helper()

	polling := ""
	if transport == "http" {
		polling = "        polling_interval: 1s\n"
	}

	config := fmt.Sprintf(`receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  otlphttp:
    endpoint: ${env:COMPAT_ENDPOINT}
extensions:
  opamp:
    server:
      %s:
        endpoint: %s://host.docker.internal:%d/api/v1/opamp
%s    instance_uid: %s
    reports_raw_config: true
    agent_description:
      include_resource_attributes: true
service:
  extensions: [opamp]
  telemetry:
    resource:
      attributes:
        - name: service.namespace
          value: compatibility
        - name: service.instance.id
          value: %s
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlphttp]
`, transport, transport, port, polling, uid, serviceID)

	return startCurrentContainer(t, testcontainers.ContainerRequest{
		Image:      currentCollectorImage,
		Files:      []testcontainers.ContainerFile{configContainerFile("/etc/collector.yaml", config)},
		Cmd:        []string{"--config=/etc/collector.yaml"},
		Env:        map[string]string{"COMPAT_ENDPOINT": "http://backend.invalid:4318"},
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
		WaitingFor: wait.ForLog("Everything is ready").WithStartupTimeout(currentClientTimeout),
	})
}

func assertCurrentEndpoints(t *testing.T, server *testutil.APIServer, uid uuid.UUID) {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/namespaces/compatibility/agents/%s/endpoints", server.Endpoint, uid)
	for range 10 {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)
		request.Header.Set("Authorization", "Bearer "+server.IssueTokenForEmail(server.Settings.Security.AdminSettings.Email))
		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		require.NoError(t, response.Body.Close())
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode, string(body))
		assert.Contains(t, string(body), "http://backend.invalid:4318")
		assert.NotContains(t, string(body), "${env:")
	}
}

func TestE2E_CurrentSupervisor(t *testing.T) {
	requireCurrentClients(t)
	t.Parallel()

	for _, transport := range []string{"ws", "http"} {
		t.Run(transport, func(t *testing.T) {
			t.Parallel()
			base := testutil.NewBase(t)
			server := base.StartStandaloneAPIServer()
			server.WaitForReady()
			apiClient := server.Client()
			uid := uuid.New()
			supervisor := startCurrentSupervisor(t, server.Port, uid, transport)
			registered := testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool {
				return a.Status.ComponentHealth.Healthy && a.Status.EffectiveConfig.ConfigMap.ConfigMap["raw"].Body != ""
			}, currentClientTimeout, conformancePoll, "Supervisor should forward the complete effective config map")
			assert.NotZero(t, uint64(registered.Metadata.Capabilities)&
				uint64(protobufs.AgentCapabilities_AgentCapabilities_ReportsRemoteConfig))
			assert.NotEmpty(t, registered.Status.EffectiveConfig.ConfigMap.ConfigMap[""].Body)
			t.Log("Supervisor connected with healthy Collector and named effective config")
			group := createSupervisorConfig(t, apiClient)
			applied := awaitConfigSync(t, apiClient, uid, "Applied")
			require.Equal(t, applied.Status.RemoteConfig.DesiredHash, applied.Status.RemoteConfig.ReportedHash)
			t.Log("First remote config acknowledged as applied")

			workingHash := bytes.Clone(applied.Status.RemoteConfig.ReportedHash)
			// This passes config validation but cannot bind: a runtime failure must restore the last working config.
			updateSupervisorConfig(t, apiClient, group,
				"receivers:\n  otlp:\n    protocols:\n      grpc:\n        endpoint: 192.0.2.1:4317\n")
			rolledBack := awaitConfigSync(t, apiClient, uid, "RolledBack")
			t.Log("Runtime failure rolled back to the working hash")
			require.Equal(t, workingHash, rolledBack.Status.RemoteConfig.ReportedHash)
			require.Equal(t, rolledBack.Status.RemoteConfig.DesiredHash, rolledBack.Status.RemoteConfig.LastFailedHash)
			require.NotEmpty(t, rolledBack.Status.RemoteConfig.LastFailedError)
			beforeRestart := testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool {
				return a.Status.ComponentHealth.Healthy && a.Status.LastReportedAt != rolledBack.Status.LastReportedAt
			}, currentClientTimeout, conformancePoll, "heartbeats should continue after rollback")
			// Reconnect using the same persistent Supervisor state; failed desired config must stay suppressed.
			require.NoError(t, supervisor.Stop(t.Context(), nil))
			require.NoError(t, supervisor.Start(t.Context()))
			reconnected := testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool {
				return a.Status.Connected && a.Status.ComponentHealth.Healthy &&
					a.Status.LastReportedAt != beforeRestart.Status.LastReportedAt &&
					a.Status.ComponentHealth.StartTime.After(beforeRestart.Status.ComponentHealth.StartTime.Time) &&
					bytes.Equal(a.Status.RemoteConfig.LastFailedHash, rolledBack.Status.RemoteConfig.DesiredHash) &&
					(a.Status.RemoteConfig.SyncStatus == "Failed" || a.Status.RemoteConfig.SyncStatus == "RolledBack") &&
					!strings.Contains(a.Status.EffectiveConfig.ConfigMap.ConfigMap[""].Body, "192.0.2.1")
			}, currentClientTimeout, conformancePoll, "Supervisor should reconnect with restored config")
			// Supervisor persists FAILED for the rejected revision and may replay it on restart
			// while continuing to run the restored configuration. It need not replay APPLIED.
			assert.Equal(t, rolledBack.Status.RemoteConfig.LastFailedHash, reconnected.Status.RemoteConfig.LastFailedHash)
			// An explicit retry allows the same desired hash to fail again, with a new failure timestamp.
			require.Eventually(t, func() bool {
				_, err := apiClient.AgentService.UpdateAgent(t.Context(), "default", uid, &v1.Agent{
					Spec: v1.AgentSpec{RetryRemoteConfig: true},
				})

				return err == nil
			}, currentClientTimeout, conformancePoll)
			testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool {
				return a.Status.RemoteConfig.SyncStatus == "RolledBack" &&
					a.Status.RemoteConfig.LastFailedAt != reconnected.Status.RemoteConfig.LastFailedAt
			}, currentClientTimeout, conformancePoll, "explicit retry should produce another failed attempt")
			updateSupervisorConfig(t, apiClient, group, "receivers:\n  unknown_component: {}\n")
			testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool {
				return bytes.Equal(a.Status.RemoteConfig.DesiredHash, a.Status.RemoteConfig.LastFailedHash) &&
					a.Status.RemoteConfig.LastFailedError != ""
			}, currentClientTimeout, conformancePoll, "invalid configuration should be acknowledged as failed")
			updateSupervisorConfig(t, apiClient, group, "processors:\n  batch:\n    timeout: 2s\n")
			awaitConfigSync(t, apiClient, uid, "Applied")
			verifySupervisorOwnMetrics(t, apiClient, uid)
		})
	}
}

func startCurrentSupervisor(t *testing.T, port int, uid uuid.UUID, transport string) testcontainers.Container {
	t.Helper()

	config := fmt.Sprintf(`server:
  endpoint: %s://host.docker.internal:%d/api/v1/opamp
  tls:
    insecure: true
capabilities:
  accepts_remote_config: true
  accepts_restart_command: true
  reports_own_metrics: true
  reports_effective_config: true
  reports_health: true
  reports_heartbeat: true
agent:
  executable: /otelcol-contrib
  instance_id: %s
  config_files: [/etc/collector.yaml, $REMOTE_CONFIG, $OPAMP_EXTENSION_CONFIG, $OWN_TELEMETRY_CONFIG]
  validate_config: true
  automatic_config_rollback: true
  config_apply_timeout: 30s
  bootstrap_timeout: 30s
  passthrough_logs: true
storage:
  directory: /tmp/supervisor-data
`, transport, port, uid)
	collector := `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  debug: {}
extensions:
  opamp:
    reports_raw_config: true
    agent_description:
      include_resource_attributes: true
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
`

	return startCurrentContainer(t, testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{Context: "testdata/supervisor", KeepImage: true},
		Files: []testcontainers.ContainerFile{
			configContainerFile("/etc/supervisor.yaml", config), configContainerFile("/etc/collector.yaml", collector),
		},
		Cmd:        []string{"--config=/etc/supervisor.yaml"},
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
		WaitingFor: wait.ForLog("Everything is ready").WithStartupTimeout(currentClientTimeout),
	})
}

func configContainerFile(path, content string) testcontainers.ContainerFile {
	return testcontainers.ContainerFile{Reader: strings.NewReader(content), ContainerFilePath: path, FileMode: 0o644}
}

func startCurrentContainer(t *testing.T, request testcontainers.ContainerRequest) testcontainers.Container {
	t.Helper()

	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: request, Started: true,
	})
	if container != nil {
		t.Cleanup(func() {
			if t.Failed() {
				logs, logErr := container.Logs(context.Background())
				if logErr == nil {
					output, _ := io.ReadAll(logs)
					_ = logs.Close()

					t.Logf("client logs:\n%s", output)
				}
			}

			_ = container.Terminate(context.Background())
		})
	}

	require.NoError(t, err)

	return container
}

func createSupervisorConfig(t *testing.T, apiClient *client.Client) *v1.AgentGroup {
	t.Helper()

	name := "pipeline"
	group, err := apiClient.AgentGroupService.CreateAgentGroup(t.Context(), "default", &v1.AgentGroup{
		Metadata: v1.Metadata{Name: "compatibility"},
		Spec: v1.Spec{
			Selector: v1.AgentSelector{IdentifyingAttributes: map[string]string{"service.name": "otelcol-contrib"}},
			AgentConfig: &v1.AgentConfig{AgentRemoteConfigs: []v1.AgentGroupRemoteConfig{{
				AgentRemoteConfigName: &name,
				AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{
					Value: "processors:\n  batch:\n    timeout: 1s\n", ContentType: "text/yaml",
				},
			}}},
		},
	})
	require.NoError(t, err)

	return group
}

func updateSupervisorConfig(t *testing.T, apiClient *client.Client, group *v1.AgentGroup, value string) {
	t.Helper()

	require.Eventually(t, func() bool {
		latest, err := apiClient.AgentGroupService.GetAgentGroup(t.Context(), "default", group.Metadata.Name)
		if err != nil {
			return false
		}

		latest.Spec.AgentConfig.AgentRemoteConfigs[0].AgentRemoteConfigSpec.Value = value

		updated, err := apiClient.AgentGroupService.UpdateAgentGroup(t.Context(), latest)
		if err != nil {
			return false
		}

		*group = *updated

		return true
	}, currentClientTimeout, conformancePoll, "update should succeed after re-reading concurrent membership changes")
}

func awaitConfigSync(t *testing.T, apiClient *client.Client, uid uuid.UUID, syncStatus string) *v1.Agent {
	t.Helper()

	return testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool {
		return a.Status.RemoteConfig.SyncStatus == syncStatus
	}, currentClientTimeout, conformancePoll, "remote configuration should become "+syncStatus)
}

// Keep released-client lifecycle tests separate from the broad, highly parallel E2E suite.
func requireCurrentClients(t *testing.T) {
	t.Helper()

	if testing.Short() || os.Getenv("OPAMP_TEST_CURRENT_CLIENTS") != "1" {
		t.Skip("set OPAMP_TEST_CURRENT_CLIENTS=1 and use -parallel=1 for released-client compatibility tests")
	}
}

// Supervisor 0.163.0 applies own-metrics settings but does not acknowledge the bundle hash.
// OpAMP endpoint replacement is excluded until its repeated-offer reconnect loop is resolved.
func verifySupervisorOwnMetrics(t *testing.T, apiClient *client.Client, uid uuid.UUID) {
	t.Helper()

	_, err := apiClient.AgentGroupService.CreateAgentGroup(t.Context(), "default", &v1.AgentGroup{
		Metadata: v1.Metadata{Name: "compatibility-connection"},
		Spec: v1.Spec{
			Selector: v1.AgentSelector{IdentifyingAttributes: map[string]string{"service.name": "otelcol-contrib"}},
			AgentConfig: &v1.AgentConfig{ConnectionSettings: &v1.ConnectionSettings{
				OwnMetrics: v1.TelemetryConnectionSettings{DestinationEndpoint: "http://metrics.invalid:4318"},
			}},
		},
	})
	require.NoError(t, err)
	testutil.EventuallyAgent(t, apiClient, "default", uid, func(a *v1.Agent) bool {
		return a.Status.Connected && a.Status.ComponentHealth.Healthy &&
			strings.Contains(a.Status.EffectiveConfig.ConfigMap.ConfigMap[""].Body, "metrics.invalid")
	}, currentClientTimeout, conformancePoll, "own metrics settings should reach the healthy managed Collector")
}
