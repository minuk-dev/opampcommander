package agentservice_test

import (
	"context"
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
// agent the builder must advertise the server capabilities and request full state.
func TestServerToAgentBuilder_Build_AdvertisesServerCapabilities(t *testing.T) {
	t.Parallel()

	builder := newTestBuilder()
	agent := agentmodel.NewAgent(uuid.New())

	msg := builder.Build(context.Background(), agent)

	require.NotNil(t, msg)

	for _, capability := range []protobufs.ServerCapabilities{
		protobufs.ServerCapabilities_ServerCapabilities_AcceptsStatus,
		protobufs.ServerCapabilities_ServerCapabilities_OffersRemoteConfig,
		protobufs.ServerCapabilities_ServerCapabilities_AcceptsEffectiveConfig,
		protobufs.ServerCapabilities_ServerCapabilities_OffersConnectionSettings,
		protobufs.ServerCapabilities_ServerCapabilities_AcceptsConnectionSettingsRequest,
		protobufs.ServerCapabilities_ServerCapabilities_OffersPackages,
		protobufs.ServerCapabilities_ServerCapabilities_AcceptsPackagesStatus,
	} {
		assert.NotZero(t, msg.GetCapabilities()&uint64(capability),
			"server capability %v should be advertised", capability)
	}

	// A not-yet-fully-described agent must be asked to report its full state.
	assert.NotZero(t,
		msg.GetFlags()&uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState),
		"ReportFullState flag should be set for an incomplete agent")
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

	msg := builder.Build(context.Background(), agent)

	require.NotNil(t, msg.GetRemoteConfig())
	assert.NotEmpty(t, msg.GetRemoteConfig().GetConfigHash())

	configFile, ok := msg.GetRemoteConfig().GetConfig().GetConfigMap()["collector.yaml"]
	require.True(t, ok, "delivered config should contain the applied file")
	assert.Equal(t, body, configFile.GetBody())
}
