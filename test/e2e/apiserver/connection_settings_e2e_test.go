//go:build e2e

package apiserver_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// Model Supervisor 0.163.0's acceptance without status reporting over both transports.
// This verifies the server's guard, not real Supervisor endpoint/certificate rotation.
//
//nolint:tparallel // Subtests share an API client; keep its initial login sequential.
func TestE2E_ConnectionSettingsWithoutAcknowledgment(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)
	server := base.StartStandaloneAPIServer()
	server.WaitForReady()
	apiClient := server.Client()

	//nolint:paralleltest // Shared API client login and group reconciliation are sequential.
	for _, transport := range []string{"ws", "http"} {
		t.Run(transport, func(t *testing.T) {
			uid := uuid.New()
			name := "no-ack-" + transport
			_, err := apiClient.AgentGroupService.CreateAgentGroup(t.Context(), "default", &v1.AgentGroup{
				Metadata: v1.Metadata{Name: name},
				Spec: v1.Spec{
					Selector: v1.AgentSelector{IdentifyingAttributes: map[string]string{"service.name": name}},
					AgentConfig: &v1.AgentConfig{ConnectionSettings: &v1.ConnectionSettings{
						OpAMP: v1.OpAMPConnectionSettings{
							DestinationEndpoint: fmt.Sprintf("%s://localhost:%d/api/v1/opamp?replacement=1", transport, server.Port),
						},
						OwnMetrics: v1.TelemetryConnectionSettings{DestinationEndpoint: "http://metrics.invalid:4318"},
					}},
				},
			})
			require.NoError(t, err)

			for seq := uint64(1); seq <= 3; seq++ {
				message := &protobufs.AgentToServer{
					InstanceUid: uid[:], SequenceNum: seq,
					Capabilities: uint64(protobufs.AgentCapabilities_AgentCapabilities_ReportsStatus |
						protobufs.AgentCapabilities_AgentCapabilities_AcceptsOpAMPConnectionSettings |
						protobufs.AgentCapabilities_AgentCapabilities_ReportsHealth),
					AgentDescription: &protobufs.AgentDescription{IdentifyingAttributes: []*protobufs.KeyValue{
						{Key: "service.name", Value: &protobufs.AnyValue{Value: &protobufs.AnyValue_StringValue{StringValue: name}}},
					}},
					Health: &protobufs.ComponentHealth{Healthy: true, Status: fmt.Sprintf("report-%d", seq)},
				}

				var response *protobufs.ServerToAgent
				if transport == "http" {
					response = postAgentToServer(t, server.Port, message)
				} else {
					response = exchangeConnectionSettingsWebSocket(t, server.Port, message)
				}

				assert.Nil(t, response.GetErrorResponse())
				assert.Nil(t, response.GetConnectionSettings(), "reconnecting must not trigger client replacement")
			}

			agent := testutil.EventuallyAgent(t, apiClient, "default", uid, func(agent *v1.Agent) bool {
				return len(agent.Spec.ConnectionSettingsHash) > 0 && agent.Status.ComponentHealth.Status == "report-3"
			}, 10*time.Second, 100*time.Millisecond, "desired settings and subsequent health reports must be retained")
			assert.True(t, agent.Status.ComponentHealth.Healthy)
			assert.Equal(t, "pending", agent.Status.ConnectionSettings.SyncStatus)
			assert.Empty(t, agent.Status.ConnectionSettings.LastConnectionSettingsHash)
		})
	}
}

func TestE2E_ConnectionSettingsAfterClientRestart(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	for _, transport := range []testutil.ReferenceAgentTransport{
		testutil.ReferenceAgentTransportWebSocket, testutil.ReferenceAgentTransportHTTP,
	} {
		t.Run(string(transport), func(t *testing.T) {
			t.Parallel()

			base := testutil.NewBase(t)
			server := base.StartStandaloneAPIServer()
			server.WaitForReady()
			apiClient := server.Client()
			uid := uuid.New()
			name := "restart-" + string(transport)
			endpoint := "http://metrics.example:4318"
			_, err := apiClient.AgentGroupService.CreateAgentGroup(t.Context(), "default", &v1.AgentGroup{
				Metadata: v1.Metadata{Name: name},
				Spec: v1.Spec{
					Selector: v1.AgentSelector{IdentifyingAttributes: map[string]string{"service.name": name}},
					AgentConfig: &v1.AgentConfig{ConnectionSettings: &v1.ConnectionSettings{
						OwnMetrics: v1.TelemetryConnectionSettings{DestinationEndpoint: endpoint},
					}},
				},
			})
			require.NoError(t, err)

			opts := []testutil.ReferenceAgentOption{
				testutil.WithReferenceAgentUID(uid),
				testutil.WithReferenceAgentTransport(transport),
				testutil.WithReferenceAgentHeartbeat(100 * time.Millisecond),
				testutil.WithReferenceAgentIdentifyingAttributes(map[string]string{"service.name": name}),
			}
			first := base.StartReferenceAgent(server.Port, opts...)

			require.Eventually(t, func() bool { return first.OwnMetricsEndpoint() == endpoint },
				10*time.Second, 100*time.Millisecond)
			testutil.EventuallyAgent(t, apiClient, "default", uid, func(agent *v1.Agent) bool {
				return agent.Status.ConnectionSettings.SyncStatus == "applied"
			}, 10*time.Second, 100*time.Millisecond, "first client applied the settings")
			first.Stop()

			// A fresh client has the same UID but neither the old settings nor its acknowledgment.
			second := base.StartReferenceAgent(server.Port, opts...)

			require.Eventually(t, func() bool { return second.OwnMetricsEndpoint() == endpoint },
				10*time.Second, 100*time.Millisecond, "persisted APPLIED must not prevent settings recovery")
		})
	}
}

// Each exchange opens a replacement socket for the same agent identity.
func exchangeConnectionSettingsWebSocket(
	t *testing.T, port int, message *protobufs.AgentToServer,
) *protobufs.ServerToAgent {
	t.Helper()

	conn, response, err := websocket.DefaultDialer.DialContext(t.Context(),
		fmt.Sprintf("ws://localhost:%d/api/v1/opamp", port), nil)
	if response != nil {
		defer func() { _ = response.Body.Close() }()
	}

	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(10*time.Second)))

	payload, err := proto.Marshal(message)
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, append([]byte{0}, payload...)))
	kind, payload, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, websocket.BinaryMessage, kind)
	require.NotEmpty(t, payload)
	require.Zero(t, payload[0], "OpAMP WebSocket header")

	var reply protobufs.ServerToAgent
	require.NoError(t, proto.Unmarshal(payload[1:], &reply))

	return &reply
}
