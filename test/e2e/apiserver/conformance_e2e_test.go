//go:build e2e

package apiserver_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"google.golang.org/protobuf/proto"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// The OpAMP conformance suite drives the server with the upstream opamp-go client
// (testutil.ReferenceAgent) over the real wire protocol and asserts both sides of each
// round-trip: what the agent received and what the REST API reflects back.
// See docs/content/en/docs/opamp-conformance.md for the capability matrix it verifies.

const (
	conformanceTimeout = 60 * time.Second
	conformancePoll    = 500 * time.Millisecond
)

// TestE2E_Conformance_ServerCapabilities sends a single AgentToServer over the plain-HTTP
// transport and checks the ServerToAgent the server answers with.
func TestE2E_Conformance_ServerCapabilities(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, "opampcommander_e2e_conformance_caps")

	apiServer.WaitForReady()

	instanceUID := uuid.New()
	//exhaustruct:ignore
	response := postAgentToServer(t, apiServer.Port, &protobufs.AgentToServer{
		InstanceUid:  instanceUID[:],
		SequenceNum:  1,
		Capabilities: uint64(protobufs.AgentCapabilities_AgentCapabilities_ReportsStatus),
	})

	assert.Nil(t, response.GetErrorResponse(), "a well-formed report must not be rejected")
	assert.Equal(t, instanceUID[:], response.GetInstanceUid(), "response must echo the instance UID")

	offered := protobufs.ServerCapabilities(response.GetCapabilities())
	for _, capability := range []protobufs.ServerCapabilities{
		protobufs.ServerCapabilities_ServerCapabilities_AcceptsStatus,
		protobufs.ServerCapabilities_ServerCapabilities_OffersRemoteConfig,
		protobufs.ServerCapabilities_ServerCapabilities_AcceptsEffectiveConfig,
		protobufs.ServerCapabilities_ServerCapabilities_OffersPackages,
		protobufs.ServerCapabilities_ServerCapabilities_AcceptsPackagesStatus,
		protobufs.ServerCapabilities_ServerCapabilities_OffersConnectionSettings,
	} {
		assert.NotZero(t, offered&capability, "server should advertise %s", capability)
	}

	assert.Zero(t, offered&protobufs.ServerCapabilities_ServerCapabilities_AcceptsConnectionSettingsRequest,
		"connection_settings_request is not processed, so the capability must be withheld")

	// The first report carries no description, so the server must ask for full state.
	assert.NotZero(t,
		response.GetFlags()&uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState),
		"server should request full state from an agent it knows nothing about")
}

func TestE2E_Conformance_ReferenceAgent(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, "opampcommander_e2e_conformance")

	apiServer.WaitForReady()

	apiClient := apiServer.Client()

	// Sequential on purpose: concurrent first logins race in the RBAC policy sync.
	//nolint:paralleltest // see above.
	for _, transport := range []testutil.ReferenceAgentTransport{
		testutil.ReferenceAgentTransportWebSocket,
		testutil.ReferenceAgentTransportHTTP,
	} {
		t.Run(string(transport), func(t *testing.T) {
			runReferenceAgentConformance(t, base, apiServer.Port, apiClient, transport)
		})
	}
}

func runReferenceAgentConformance(
	t *testing.T,
	base *testutil.Base,
	opampPort int,
	apiClient *client.Client,
	transport testutil.ReferenceAgentTransport,
) {
	t.Helper()

	ctx := t.Context()
	serviceName := "conformance-" + string(transport)

	agent := base.StartReferenceAgent(opampPort,
		testutil.WithReferenceAgentTransport(transport),
		testutil.WithReferenceAgentIdentifyingAttributes(map[string]string{"service.name": serviceName}),
		testutil.WithReferenceAgentNonIdentifyingAttributes(map[string]string{"host.name": "conformance-host"}),
	)

	t.Run("connect and AgentDescription ingest", func(t *testing.T) {
		registered := testutil.EventuallyAgent(t, apiClient, "default", agent.UID, func(a *v1.Agent) bool {
			return a.Metadata.Capabilities != 0 && len(a.Metadata.Description.IdentifyingAttributes) > 0
		}, conformanceTimeout, conformancePoll, "agent should register with its description and capabilities")

		assert.Equal(t, "default", registered.Metadata.Namespace)
		assert.Equal(t, serviceName, registered.Metadata.Description.IdentifyingAttributes["service.name"])
		assert.Equal(t, "conformance-host", registered.Metadata.Description.NonIdentifyingAttributes["host.name"])
		assert.Equal(t, v1.AgentCapabilities(testutil.ReferenceAgentCapabilities), registered.Metadata.Capabilities)
		assert.True(t, registered.Status.Connected)
		assert.Equal(t, transportConnectionType(transport), registered.Status.ConnectionType)
	})

	t.Run("health reporting", func(t *testing.T) {
		testutil.EventuallyAgent(t, apiClient, "default", agent.UID, func(a *v1.Agent) bool {
			return a.Status.ComponentHealth.Healthy &&
				a.Status.ComponentHealth.Status == "StatusOK" &&
				a.Status.ComponentHealth.ComponentsMap["pipeline:traces"] == "StatusOK"
		}, conformanceTimeout, conformancePoll, "reported health should surface on the agent resource")

		require.NoError(t, agent.ReportHealth(false, "StatusRecoverableError"))

		testutil.EventuallyAgent(t, apiClient, "default", agent.UID, func(a *v1.Agent) bool {
			return !a.Status.ComponentHealth.Healthy &&
				a.Status.ComponentHealth.Status == "StatusRecoverableError"
		}, conformanceTimeout, conformancePoll, "a health change should surface on the agent resource")

		require.NoError(t, agent.ReportHealth(true, "StatusOK"))
	})

	t.Run("remote config and effective config round-trip", func(t *testing.T) {
		groupName := serviceName + "-config"
		configName := "pipeline"
		configBody := "receivers:\n  otlp: {}\n# " + serviceName + "\n"

		//exhaustruct:ignore
		_, err := apiClient.AgentGroupService.CreateAgentGroup(ctx, "default", &v1.AgentGroup{
			Metadata: v1.Metadata{Name: groupName},
			Spec: v1.Spec{
				Selector: v1.AgentSelector{
					IdentifyingAttributes: map[string]string{"service.name": serviceName},
				},
				AgentConfig: &v1.AgentConfig{
					AgentRemoteConfigs: []v1.AgentGroupRemoteConfig{{
						AgentRemoteConfigName: &configName,
						AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{
							Value:       configBody,
							ContentType: "text/yaml",
						},
					}},
				},
			},
		})
		require.NoError(t, err)

		configKey := groupName + "/" + configName

		require.Eventually(t, func() bool {
			return agent.EffectiveConfig()[configKey] == configBody
		}, conformanceTimeout, conformancePoll, "agent should receive and apply the remote config")

		testutil.EventuallyAgent(t, apiClient, "default", agent.UID, func(a *v1.Agent) bool {
			return a.Status.EffectiveConfig.ConfigMap.ConfigMap[configKey].Body == configBody
		}, conformanceTimeout, conformancePoll, "the applied config should be reported back as the effective config")

		assert.Equal(t, 1, agent.RemoteConfigsApplied(), "one config change should be one distinct offer")
	})

	t.Run("connection settings offer", func(t *testing.T) {
		groupName := serviceName + "-connection"
		endpoint := "http://metrics.conformance.invalid:4318/" + serviceName

		//exhaustruct:ignore
		_, err := apiClient.AgentGroupService.CreateAgentGroup(ctx, "default", &v1.AgentGroup{
			Metadata: v1.Metadata{Name: groupName},
			Spec: v1.Spec{
				Selector: v1.AgentSelector{
					IdentifyingAttributes: map[string]string{"service.name": serviceName},
				},
				AgentConfig: &v1.AgentConfig{
					//exhaustruct:ignore
					ConnectionSettings: &v1.ConnectionSettings{
						//exhaustruct:ignore
						OwnMetrics: v1.TelemetryConnectionSettings{DestinationEndpoint: endpoint},
					},
				},
			},
		})
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			return agent.OwnMetricsEndpoint() == endpoint
		}, conformanceTimeout, conformancePoll, "agent should receive the own-metrics connection offer")
	})

	t.Run("restart command round-trip", func(t *testing.T) {
		// The agent's own reports race the update; a 409 asks the caller to retry.
		require.Eventually(t, func() bool {
			_, err := apiClient.AgentService.RestartAgent(ctx, "default", agent.UID)

			return err == nil
		}, conformanceTimeout, conformancePoll, "restart request should be accepted")

		if !assert.Eventually(t, func() bool {
			return agent.Restarts() == 1
		}, conformanceTimeout, conformancePoll, "agent should receive the Restart command") {
			observed, err := apiClient.AgentService.GetAgent(ctx, "default", agent.UID)
			require.NoError(t, err)
			t.Fatalf("last observed agent:\n%s", testutil.DumpJSON(t, observed))
		}

		// Once the agent reports a start time after the request, the command is satisfied
		// and must not be sent again.
		time.Sleep(3 * time.Second)
		assert.Equal(t, 1, agent.Restarts(), "a satisfied restart must not be re-sent")
	})

	t.Run("package status reporting", func(t *testing.T) {
		require.NoError(t, agent.ReportPackageStatuses(map[string]string{"otelcol-contrib": "0.115.1"}))

		testutil.EventuallyAgent(t, apiClient, "default", agent.UID, func(a *v1.Agent) bool {
			_, ok := a.Status.PackageStatuses.Packages["otelcol-contrib"]

			return ok
		}, conformanceTimeout, conformancePoll, "reported package statuses should surface on the agent resource")
	})

	t.Run("packages available offer", func(t *testing.T) {
		t.Skip("not reachable through the public API yet: nothing populates an agent's " +
			"spec.packagesAvailable (see 'Known gaps' in docs/content/en/docs/opamp-conformance.md)")
	})

	assert.Empty(t, agent.ServerErrors(), "a conformant agent should never receive an error_response")
}

// TestE2E_Conformance_OTelCollector is the interop check against a real agent: the
// otelcol-contrib opamp extension (see the interoperability matrix in the README).
func TestE2E_Conformance_OTelCollector(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, "opampcommander_e2e_conformance_otelcol")

	apiServer.WaitForReady()

	collector := base.StartOTelCollector(apiServer.Port)

	apiClient := apiServer.Client()

	registered := testutil.EventuallyAgent(t, apiClient, "default", collector.UID, func(a *v1.Agent) bool {
		return a.Metadata.Capabilities != 0 &&
			len(a.Metadata.Description.IdentifyingAttributes) > 0 &&
			len(a.Status.EffectiveConfig.ConfigMap.ConfigMap) > 0
	}, conformanceTimeout, conformancePoll, "collector should register with its description, capabilities and effective config")

	assert.Equal(t, "otelcol-contrib", registered.Metadata.Type)
	assert.True(t, registered.Status.Connected)

	effective := strings.Join(configBodies(registered.Status.EffectiveConfig), "\n")
	assert.Contains(t, effective, "opamp", "effective config should be the collector's running config")
}

func transportConnectionType(transport testutil.ReferenceAgentTransport) string {
	if transport == testutil.ReferenceAgentTransportHTTP {
		return "HTTP"
	}

	return "WebSocket"
}

func configBodies(config v1.AgentEffectiveConfig) []string {
	bodies := make([]string, 0, len(config.ConfigMap.ConfigMap))
	for _, file := range config.ConfigMap.ConfigMap {
		bodies = append(bodies, file.Body)
	}

	return bodies
}

// postAgentToServer performs one exchange over the OpAMP plain-HTTP transport.
func postAgentToServer(t *testing.T, port int, message *protobufs.AgentToServer) *protobufs.ServerToAgent {
	t.Helper()

	body, err := proto.Marshal(message)
	require.NoError(t, err)

	url := fmt.Sprintf("http://localhost:%d/api/v1/opamp", port)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/x-protobuf")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	defer func() { _ = response.Body.Close() }()

	require.Equal(t, http.StatusOK, response.StatusCode)

	payload, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	var serverToAgent protobufs.ServerToAgent
	require.NoError(t, proto.Unmarshal(payload, &serverToAgent))

	return &serverToAgent
}
