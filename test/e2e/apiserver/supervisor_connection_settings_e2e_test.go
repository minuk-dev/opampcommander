//go:build e2e

package apiserver_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
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

const supervisorPolicyTimeout = 120 * time.Second

// This is a released Supervisor, not the reference client. Rotation is unsupported:
// verify safe withholding, continued health, unchanged credentials and telemetry recovery.
//
//nolint:paralleltest // Run one Supervisor/Collector pair at a time on small Docker hosts.
func TestE2E_SupervisorConnectionSettingsPolicy(t *testing.T) {
	if testing.Short() || os.Getenv("OPAMP_TEST_SUPERVISOR_POLICY") != "1" {
		t.Skip("set OPAMP_TEST_SUPERVISOR_POLICY=1 to run the released Supervisor policy regression")
	}

	//nolint:paralleltest // Run one Supervisor/Collector pair at a time on small Docker hosts.
	for _, transport := range []string{"wss", "https"} {
		t.Run(transport, func(t *testing.T) {
			base := testutil.NewBase(t)
			settings, roots, ca, caKey := mtlsFixture(t)
			server := base.StartStandaloneAPIServerWithOpAMPTLS(settings)
			require.Eventually(t, server.IsReady, 30*time.Second, 100*time.Millisecond)
			apiClient := client.New(server.Endpoint,
				client.WithTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}),
				client.WithBasicAuth(server.AdminUsername(), server.AdminPassword()))
			uid := uuid.New()
			register := func(name string) *v1.Certificate {
				cert, err := apiClient.CertificateService.CreateCertificate(t.Context(), "default", &v1.Certificate{
					Metadata: v1.CertificateMetadata{Name: name, Namespace: "default"},
					Spec:     mtlsClientCertificate(t, ca, caKey, uid),
				})
				require.NoError(t, err)

				return cert
			}
			oldCert := register("supervisor-original")
			newCert := register("supervisor-replacement")
			endpoint := fmt.Sprintf("%s://host.docker.internal:%d/api/v1/opamp", transport, server.Port)
			supervisor := startPolicySupervisor(t, endpoint, uid, settings.CAFile, oldCert)
			registered := testutil.EventuallyAgent(t, apiClient, "default", uid, func(agent *v1.Agent) bool {
				return agent.Status.Connected && agent.Status.ComponentHealth.Healthy
			}, supervisorPolicyTimeout, conformancePoll, "released Supervisor should report a healthy Collector")
			assert.NotZero(t, uint64(registered.Metadata.Capabilities)&
				uint64(protobufs.AgentCapabilities_AgentCapabilities_AcceptsOpAMPConnectionSettings))
			assert.Zero(t, uint64(registered.Metadata.Capabilities)&
				uint64(protobufs.AgentCapabilities_AgentCapabilities_ReportsConnectionSettingsStatus))
			assert.Equal(t, 1, strings.Count(policySupervisorLogs(t, supervisor), "Connecting to OpAMP server..."))

			group, err := apiClient.AgentGroupService.CreateAgentGroup(t.Context(), "default", &v1.AgentGroup{
				Metadata: v1.Metadata{Name: "supervisor-policy"},
				Spec: v1.Spec{
					Selector: v1.AgentSelector{IdentifyingAttributes: map[string]string{"service.name": "otelcol-contrib"}},
					AgentConfig: &v1.AgentConfig{ConnectionSettings: &v1.ConnectionSettings{
						OpAMP:      v1.OpAMPConnectionSettings{DestinationEndpoint: endpoint + "?replacement=1"},
						OwnMetrics: v1.TelemetryConnectionSettings{DestinationEndpoint: "http://withheld.invalid:4318"},
					}},
				},
			})
			require.NoError(t, err)
			assertSupervisorWithheld(t, apiClient, uid)
			t.Log("endpoint offer withheld across subsequent reports; Collector remains healthy")
			assert.Equal(t, 1, strings.Count(policySupervisorLogs(t, supervisor), "Connecting to OpAMP server..."),
				"repeated reports must not replace the OpAMP client")

			group.Spec.AgentConfig.ConnectionSettings.OpAMP.CertificateName = &newCert.Metadata.Name
			patchConnectionSettings(t, apiClient, group)
			assertSupervisorWithheld(t, apiClient, uid)
			t.Log("certificate offer withheld across subsequent reports; Collector remains healthy")
			assert.Equal(t, 1, strings.Count(policySupervisorLogs(t, supervisor), "Connecting to OpAMP server..."),
				"certificate offers must also be withheld")

			before, err := apiClient.AgentService.GetAgent(t.Context(), "default", uid)
			require.NoError(t, err)
			require.NoError(t, supervisor.Stop(t.Context(), nil))
			require.NoError(t, supervisor.Start(t.Context()))
			testutil.EventuallyAgent(t, apiClient, "default", uid, func(agent *v1.Agent) bool {
				return agent.Status.Connected && agent.Status.ComponentHealth.Healthy &&
					agent.Status.ComponentHealth.StartTime.After(before.Status.ComponentHealth.StartTime.Time)
			}, supervisorPolicyTimeout, conformancePoll, "original certificate and UID must still work after process restart")
			assertSupervisorWithheld(t, apiClient, uid)
			t.Log("Supervisor restarted with its original certificate and UID; policy remains effective")
			assert.Equal(t, 2, strings.Count(policySupervisorLogs(t, supervisor), "Connecting to OpAMP server..."))

			// Removing the unsupported OpAMP part must unblock the full telemetry bundle.
			group.Spec.AgentConfig.ConnectionSettings.OpAMP = v1.OpAMPConnectionSettings{}
			for _, destination := range []string{"http://first.invalid:4318", "http://second.invalid:4318"} {
				group.Spec.AgentConfig.ConnectionSettings.OwnMetrics.DestinationEndpoint = destination
				patchConnectionSettings(t, apiClient, group)
				applied := testutil.EventuallyAgent(t, apiClient, "default", uid, func(agent *v1.Agent) bool {
					return agent.Status.ComponentHealth.Healthy &&
						strings.Contains(agent.Status.EffectiveConfig.ConfigMap.ConfigMap[""].Body, destination)
				}, supervisorPolicyTimeout, conformancePoll, "new settings must reach the restarted managed Collector")
				assert.Equal(t, "pending", applied.Status.ConnectionSettings.SyncStatus)
				assert.Empty(t, applied.Status.ConnectionSettings.LastConnectionSettingsHash)
				t.Logf("managed Collector received new telemetry settings: %s", destination)
			}
		})
	}
}

func assertSupervisorWithheld(t *testing.T, apiClient *client.Client, uid uuid.UUID) {
	t.Helper()

	before, err := apiClient.AgentService.GetAgent(t.Context(), "default", uid)
	require.NoError(t, err)

	healthy := true
	observed := testutil.EventuallyAgent(t, apiClient, "default", uid, func(agent *v1.Agent) bool {
		healthy = healthy && agent.Status.ComponentHealth.Healthy

		return len(agent.Spec.ConnectionSettingsHash) > 0 && agent.Status.SequenceNum >= before.Status.SequenceNum+2
	}, supervisorPolicyTimeout, conformancePoll, "at least two subsequent Supervisor reports must complete")
	assert.True(t, healthy, "health must remain healthy throughout withheld offers")
	assert.Equal(t, "pending", observed.Status.ConnectionSettings.SyncStatus)
	assert.Empty(t, observed.Status.ConnectionSettings.LastConnectionSettingsHash)
	assert.NotContains(t, observed.Status.EffectiveConfig.ConfigMap.ConfigMap[""].Body, "withheld.invalid")
}

func startPolicySupervisor(
	t *testing.T, endpoint string, uid uuid.UUID, caFile string, certificate *v1.Certificate,
) testcontainers.Container {
	t.Helper()

	config := fmt.Sprintf(`server:
  endpoint: %s
  tls:
    ca_file: /etc/ca.pem
    cert_file: /etc/client.pem
    key_file: /etc/client.key
    server_name_override: localhost
capabilities:
  accepts_opamp_connection_settings: true
  accepts_remote_config: true
  reports_own_metrics: true
  reports_effective_config: true
  reports_health: true
  reports_heartbeat: true
agent:
  executable: /otelcol-contrib
  instance_id: %s
  config_files: [/etc/collector.yaml, $REMOTE_CONFIG, $OPAMP_EXTENSION_CONFIG, $OWN_TELEMETRY_CONFIG]
  validate_config: true
  config_apply_timeout: 60s
  bootstrap_timeout: 60s
  passthrough_logs: true
storage:
  directory: /tmp/supervisor-data
telemetry:
  logs:
    level: debug
`, endpoint, uid)
	collector := `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  debug: {}
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
`
	file := func(path, contents string) testcontainers.ContainerFile {
		return testcontainers.ContainerFile{Reader: strings.NewReader(contents), ContainerFilePath: path, FileMode: 0o644}
	}
	ca, err := os.ReadFile(caFile) //nolint:gosec // Path comes from this test's generated CA fixture.
	require.NoError(t, err)

	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{Context: "testdata/supervisor", KeepImage: true},
			Files: []testcontainers.ContainerFile{
				file("/etc/supervisor.yaml", config), file("/etc/collector.yaml", collector),
				file("/etc/ca.pem", string(ca)), file("/etc/client.pem", certificate.Spec.Cert),
				file("/etc/client.key", certificate.Spec.PrivateKey),
			},
			Cmd:        []string{"--config=/etc/supervisor.yaml"},
			ExtraHosts: []string{"host.docker.internal:host-gateway"},
			WaitingFor: wait.ForLog("Everything is ready").WithStartupTimeout(supervisorPolicyTimeout),
		},
		Started: true,
	})
	if container != nil {
		t.Cleanup(func() {
			if t.Failed() {
				t.Log(policySupervisorLogs(t, container))
			}

			_ = container.Terminate(context.Background())
		})
	}

	require.NoError(t, err)

	return container
}

func policySupervisorLogs(t *testing.T, container testcontainers.Container) string {
	t.Helper()

	logs, err := container.Logs(context.Background())
	require.NoError(t, err)

	defer func() { _ = logs.Close() }()

	output, err := io.ReadAll(logs)
	require.NoError(t, err)

	return string(output)
}
