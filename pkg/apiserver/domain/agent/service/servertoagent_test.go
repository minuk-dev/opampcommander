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

func newTestBuilder() *agentservice.ServerToAgentBuilder {
	return agentservice.NewServerToAgentBuilder(nil, slog.Default())
}

// agentWithPackages returns an agent advertising the given package names. It carries the
// AcceptsPackages capability so HasNewPackages() (and thus the package offer) is enabled.
func agentWithPackages(names ...string) *agentmodel.Agent {
	capabilities := modelagent.Capabilities(modelagent.AgentCapabilityAcceptsPackages)
	agent := agentmodel.NewAgent(uuid.New(), agentmodel.WithCapabilities(&capabilities))
	agent.Spec.PackagesAvailable = &agentmodel.AgentSpecPackage{Packages: names}

	return agent
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

// TestServerToAgentBuilder_Build_IncludesConnectionSettings exercises the domain→protobuf
// connection-settings conversion path (opamp/own_metrics/own_logs/own_traces/other_connections,
// each with headers + TLS certificate) end-to-end through the builder, so the offer the agent
// receives carries every configured destination, header and certificate.
func TestServerToAgentBuilder_Build_IncludesConnectionSettings(t *testing.T) {
	t.Parallel()

	cert := &agentmodel.AgentCertificate{
		Cert:       []byte("cert"),
		PrivateKey: []byte("key"),
		CaCert:     []byte("ca"),
	}

	connectionInfo, err := agentmodel.NewConnectionInfo(
		&agentmodel.AgentOpAMPConnectionSettings{
			DestinationEndpoint: "wss://opamp.example/v1/opamp",
			Headers:             map[string][]string{"Authorization": {"Bearer token"}},
			Certificate:         cert,
		},
		&agentmodel.AgentTelemetryConnectionSettings{
			DestinationEndpoint: "https://metrics.example/v1/metrics",
			Headers:             nil,
			Certificate:         nil,
		},
		nil,
		nil,
		map[string]agentmodel.AgentOtherConnectionSettings{
			"custom": {
				DestinationEndpoint: "https://other.example",
				Headers:             nil,
				Certificate:         nil,
			},
		},
	)
	require.NoError(t, err)

	agent := agentmodel.NewAgent(uuid.New())
	agent.Spec.ConnectionInfo = connectionInfo

	msg := newTestBuilder().Build(t.Context(), agent)

	offers := msg.GetConnectionSettings()
	require.NotNil(t, offers, "connection settings should be offered")
	assert.NotEmpty(t, offers.GetHash())

	opamp := offers.GetOpamp()
	require.NotNil(t, opamp)
	assert.Equal(t, "wss://opamp.example/v1/opamp", opamp.GetDestinationEndpoint())
	require.NotNil(t, opamp.GetCertificate())
	assert.Equal(t, []byte("cert"), opamp.GetCertificate().GetCert())

	headers := opamp.GetHeaders().GetHeaders()
	require.Len(t, headers, 1)
	assert.Equal(t, "Authorization", headers[0].GetKey())
	assert.Equal(t, "Bearer token", headers[0].GetValue())

	require.NotNil(t, offers.GetOwnMetrics())
	assert.Equal(t, "https://metrics.example/v1/metrics", offers.GetOwnMetrics().GetDestinationEndpoint())
	// A telemetry setting without a certificate must not synthesize an empty TLS certificate.
	assert.Nil(t, offers.GetOwnMetrics().GetCertificate())

	require.Contains(t, offers.GetOtherConnections(), "custom")
	assert.Equal(t, "https://other.example", offers.GetOtherConnections()["custom"].GetDestinationEndpoint())
}

// Even stale or internally injected assignments must not emit unsupported offers.
func TestServerToAgentBuilder_Build_WithholdsPackages(t *testing.T) {
	t.Parallel()
	message := newTestBuilder().Build(t.Context(), agentWithPackages("otelcol-contrib"))
	assert.Nil(t, message.GetPackagesAvailable())
	assert.Zero(t, message.GetCapabilities()&uint64(protobufs.ServerCapabilities_ServerCapabilities_OffersPackages))
	assert.NotZero(t,
		message.GetCapabilities()&uint64(protobufs.ServerCapabilities_ServerCapabilities_AcceptsPackagesStatus))
}

// TestServerToAgentBuilder_Build_AdvertisesCustomCapabilities verifies that the server's
// registered custom capabilities are advertised in ServerToAgent.custom_capabilities, and that
// an empty set leaves the field unset (preserving the no-handlers default).
func TestServerToAgentBuilder_Build_AdvertisesCustomCapabilities(t *testing.T) {
	t.Parallel()

	t.Run("advertises the configured custom capabilities", func(t *testing.T) {
		t.Parallel()

		builder := agentservice.NewServerToAgentBuilder(
			agentservice.ServerCustomCapabilities{"com.example.a", "com.example.b"},
			slog.Default(),
		)

		msg := builder.Build(t.Context(), agentmodel.NewAgent(uuid.New()))

		require.NotNil(t, msg.GetCustomCapabilities())
		assert.Equal(t, []string{"com.example.a", "com.example.b"},
			msg.GetCustomCapabilities().GetCapabilities())
	})

	t.Run("no custom capabilities leaves the field unset", func(t *testing.T) {
		t.Parallel()

		msg := newTestBuilder().Build(t.Context(), agentmodel.NewAgent(uuid.New()))

		assert.Nil(t, msg.GetCustomCapabilities())
		assert.Nil(t, msg.GetCustomMessage())
	})
}
