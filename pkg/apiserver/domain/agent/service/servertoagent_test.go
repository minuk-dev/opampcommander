package agentservice_test

import (
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	modelagent "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
)

// newTestBuilder builds a ServerToAgentBuilder with no package usecase: the tests here do
// not exercise packages, so it is never invoked.
func newTestBuilder() *agentservice.ServerToAgentBuilder {
	return agentservice.NewServerToAgentBuilder(nil, slog.Default())
}

// TestServerToAgentBuilder_Build_AdvertisesServerCapabilities guards the regression that
// the cross-server push path used to send an almost-empty ServerToAgent. Even for a bare
// agent the builder must advertise the server capabilities.
func TestServerToAgentBuilder_Build_AdvertisesServerCapabilities(t *testing.T) {
	t.Parallel()

	builder := newTestBuilder()
	agent := agentmodel.NewAgent(uuid.New())

	msg := builder.Build(t.Context(), agent)

	require.NotNil(t, msg)

	for _, capability := range []protobufs.ServerCapabilities{
		protobufs.ServerCapabilities_ServerCapabilities_AcceptsStatus,
		protobufs.ServerCapabilities_ServerCapabilities_OffersRemoteConfig,
		protobufs.ServerCapabilities_ServerCapabilities_AcceptsEffectiveConfig,
		protobufs.ServerCapabilities_ServerCapabilities_OffersConnectionSettings,
		protobufs.ServerCapabilities_ServerCapabilities_OffersPackages,
		protobufs.ServerCapabilities_ServerCapabilities_AcceptsPackagesStatus,
	} {
		assert.NotZero(t, msg.GetCapabilities()&uint64(capability),
			"server capability %v should be advertised", capability)
	}

	// AcceptsConnectionSettingsRequest must NOT be advertised while the server does not
	// process connection_settings_request — advertising it would invite ignored requests.
	assert.Zero(t,
		msg.GetCapabilities()&uint64(protobufs.ServerCapabilities_ServerCapabilities_AcceptsConnectionSettingsRequest),
		"AcceptsConnectionSettingsRequest must not be advertised until it is handled")
}

// completeAgent returns an agent that has reported a description and capabilities, so
// IsComplete() is true.
func completeAgent(t *testing.T) *agentmodel.Agent {
	t.Helper()

	capabilities := modelagent.Capabilities(modelagent.AgentCapabilityReportsStatus)
	agent := agentmodel.NewAgent(uuid.New(), agentmodel.WithCapabilities(&capabilities))
	agent.Metadata.Description.IdentifyingAttributes = map[string]string{"service.name": "collector"}
	require.True(t, agent.Metadata.IsComplete())

	return agent
}

// TestServerToAgentBuilder_Build_ReportFullState pins the exact condition for the
// ReportFullState flag: requested only while the agent's reported info is incomplete, and
// NOT once it is complete. Setting it unconditionally previously drove an endless agent
// re-report loop.
func TestServerToAgentBuilder_Build_ReportFullState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		agent         *agentmodel.Agent
		wantFullState bool
	}{
		{"incomplete agent is asked to report full state", agentmodel.NewAgent(uuid.New()), true},
		{"complete agent is not asked", completeAgent(t), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := newTestBuilder().Build(t.Context(), tt.agent)
			set := msg.GetFlags()&uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState) != 0
			assert.Equal(t, tt.wantFullState, set)
		})
	}
}

// TestServerToAgentBuilder_Build_IncludesRemoteConfig is the core of the two-builders
// unification: a config assigned to the agent must be delivered by the shared builder, so a
// cross-server push carries the config instead of an empty message.
func TestServerToAgentBuilder_Build_IncludesRemoteConfig(t *testing.T) {
	t.Parallel()

	builder := newTestBuilder()

	capabilities := modelagent.Capabilities(modelagent.AgentCapabilityAcceptsRemoteConfig)
	agent := agentmodel.NewAgent(uuid.New(), agentmodel.WithCapabilities(&capabilities))

	body := []byte("receivers: {}\n")
	require.NoError(t, agent.ApplyRemoteConfig("collector.yaml", agentmodel.AgentConfigFile{
		Body:        body,
		ContentType: "text/yaml",
	}))

	msg := builder.Build(t.Context(), agent)

	require.NotNil(t, msg.GetRemoteConfig())
	assert.NotEmpty(t, msg.GetRemoteConfig().GetConfigHash())

	configFile, ok := msg.GetRemoteConfig().GetConfig().GetConfigMap()["collector.yaml"]
	require.True(t, ok, "delivered config should contain the applied file")
	assert.Equal(t, body, configFile.GetBody())
}
