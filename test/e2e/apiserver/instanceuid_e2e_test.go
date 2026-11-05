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
	"github.com/testcontainers/testcontainers-go"
	"google.golang.org/protobuf/proto"

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
		assert.NotEqual(t, v1.ConditionType("InstanceUIDConflict"), condition.Type)
	}

	agents, err := apiClient.AgentService.ListAgents(t.Context(), "default")
	require.NoError(t, err)
	require.Len(t, agents.Items, 1, "restart must not split the agent into another record")
	assert.Equal(t, uid, agents.Items[0].Metadata.InstanceUID)
}

func TestE2E_Conformance_OverlappingInstanceUIDSessions(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	server := base.StartStandaloneAPIServer()
	server.WaitForReady()
	apiClient := server.Client()
	uid := uuid.New()
	owner := dialIdentitySession(t, server.Port)
	contender := dialIdentitySession(t, server.Port)

	report := func(serviceID string) *protobufs.AgentToServer {
		return &protobufs.AgentToServer{
			InstanceUid: uid[:],
			AgentDescription: &protobufs.AgentDescription{
				IdentifyingAttributes: []*protobufs.KeyValue{{
					Key:   "service.instance.id",
					Value: &protobufs.AnyValue{Value: &protobufs.AnyValue_StringValue{StringValue: serviceID}},
				}},
			},
		}
	}

	require.Nil(t, exchangeIdentityMessage(t, owner, report("before-update")).GetErrorResponse())
	require.Nil(t, exchangeIdentityMessage(t, owner, report("after-update")).GetErrorResponse(),
		"the owning session can change its description")
	before, err := apiClient.AgentService.GetAgent(t.Context(), "default", uid)
	require.NoError(t, err)
	require.Equal(t, "after-update", before.Metadata.Description.IdentifyingAttributes["service.instance.id"])

	for _, message := range []*protobufs.AgentToServer{
		report("after-update"), // A clone must not take over by copying the description.
		report("after-restart"),
		{InstanceUid: uid[:]}, // Omitting the description must not bypass ownership.
	} {
		reply := exchangeIdentityMessage(t, contender, message)
		require.NotNil(t, reply.GetErrorResponse())
		assert.Equal(t, protobufs.ServerErrorResponseType_ServerErrorResponseType_Unavailable,
			reply.GetErrorResponse().GetType())
		assert.Equal(t, uid[:], reply.GetInstanceUid())
		assert.Nil(t, reply.GetAgentIdentification(), "a reconnect must not be assigned another UID")
	}

	during, err := apiClient.AgentService.GetAgent(t.Context(), "default", uid)
	require.NoError(t, err)
	assert.Equal(t, before.Metadata.Description, during.Metadata.Description)
	assert.Equal(t, before.Spec, during.Spec)
	require.Nil(t, exchangeIdentityMessage(t, owner, report("still-owner")).GetErrorResponse(),
		"rejected sessions must not displace the owner")

	require.NoError(t, owner.Close())
	require.Eventually(t, func() bool {
		reply := exchangeIdentityMessage(t, contender, report("after-restart"))
		assert.Nil(t, reply.GetAgentIdentification())

		return reply.GetErrorResponse() == nil
	}, 10*time.Second, 100*time.Millisecond, "overlapping reconnect should recover under the original UID")

	after, err := apiClient.AgentService.GetAgent(t.Context(), "default", uid)
	require.NoError(t, err)
	assert.Equal(t, "after-restart", after.Metadata.Description.IdentifyingAttributes["service.instance.id"])
	assert.Equal(t, before.Spec, after.Spec)
	agents, err := apiClient.AgentService.ListAgents(t.Context(), "default")
	require.NoError(t, err)
	require.Len(t, agents.Items, 1)
	assert.Equal(t, uid, agents.Items[0].Metadata.InstanceUID)
}

func dialIdentitySession(t *testing.T, port int) *websocket.Conn {
	t.Helper()

	conn, response, err := websocket.DefaultDialer.DialContext(t.Context(),
		fmt.Sprintf("ws://localhost:%d/api/v1/opamp", port), nil)
	if response != nil {
		defer func() { _ = response.Body.Close() }()
	}

	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

func exchangeIdentityMessage(t *testing.T, conn *websocket.Conn, message *protobufs.AgentToServer) *protobufs.ServerToAgent {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(10*time.Second)))
	payload, err := proto.Marshal(message)
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, append([]byte{0}, payload...)))
	kind, payload, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, websocket.BinaryMessage, kind)
	require.NotEmpty(t, payload)
	require.Zero(t, payload[0])

	var reply protobufs.ServerToAgent
	require.NoError(t, proto.Unmarshal(payload[1:], &reply))

	return &reply
}
