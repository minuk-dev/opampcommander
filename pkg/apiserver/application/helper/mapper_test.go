package helper_test

import (
	"crypto/sha256"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

func TestMapAgentConnectionSettingsSyncStatus(t *testing.T) {
	t.Parallel()

	agent := agentmodel.NewAgent(uuid.New())
	require.NoError(t, agent.ApplyConnectionSettings(&agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: "wss://example.test/api/v1/opamp",
	}, nil, nil, nil, nil))

	mapper := helper.NewMapper(clock.RealClock{}, 0)

	assert.Equal(t, "pending", mapper.MapAgentToAPI(agent).Status.ConnectionSettings.SyncStatus)
	agent.Status.ConnectionSettingsStatus.Status = agentmodel.ConnectionSettingsStatusApplied
	agent.Status.ConnectionSettingsStatus.LastConnectionSettingsHash = agent.Spec.ConnectionInfo.Hash.Bytes()
	assert.Equal(t, "applied", mapper.MapAgentToAPI(agent).Status.ConnectionSettings.SyncStatus)

	agent.Status.ConnectionSettingsStatus.Status = agentmodel.ConnectionSettingsStatusFailed
	assert.Equal(t, "failed", mapper.MapAgentToAPI(agent).Status.ConnectionSettings.SyncStatus)

	require.NoError(t, agent.ApplyConnectionSettings(&agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: "wss://next.example.test/api/v1/opamp",
	}, nil, nil, nil, nil))
	assert.Equal(t, "pending", mapper.MapAgentToAPI(agent).Status.ConnectionSettings.SyncStatus)
}

func TestMapAgentRemoteConfigSchemaRefsSource(t *testing.T) {
	t.Parallel()

	mapper := helper.NewMapper(clock.RealClock{}, 0)

	for _, tt := range []struct {
		name   string
		source agentmodel.SchemaRefsSource
	}{
		{name: "legacy"},
		{name: "automatic", source: agentmodel.SchemaRefsSourceAuto},
		{name: "explicit", source: agentmodel.SchemaRefsSourceExplicit},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mapped := mapper.MapAgentRemoteConfigToAPI(&agentmodel.AgentRemoteConfig{
				Status: agentmodel.AgentRemoteConfigResourceStatus{SchemaRefsSource: tt.source},
			})
			assert.Equal(t, string(tt.source), mapped.Status.SchemaRefsSource)
			data, err := json.Marshal(mapped.Status)
			require.NoError(t, err)

			if tt.source == "" {
				assert.NotContains(t, string(data), "schemaRefsSource")
			} else {
				assert.Contains(t, string(data), `"schemaRefsSource":"`+string(tt.source)+`"`)
			}

			assert.Empty(t, mapper.MapAPIToAgentRemoteConfig(mapped).Status.SchemaRefsSource,
				"client status must not be accepted on writes")
		})
	}
}

func TestMapAgentConnectionSettingsHashBelongsToSpec(t *testing.T) {
	t.Parallel()

	agent := agentmodel.NewAgent(uuid.New())
	require.NoError(t, agent.ApplyConnectionSettings(&agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: "wss://example.test/api/v1/opamp",
	}, nil, nil, nil, nil))

	mapped := helper.NewMapper(clock.RealClock{}, 0).MapAgentToAPI(agent)
	assert.Equal(t, agent.Spec.ConnectionInfo.Hash.Bytes(), mapped.Spec.ConnectionSettingsHash)

	data, err := json.Marshal(mapped)
	require.NoError(t, err)

	var response map[string]map[string]any
	require.NoError(t, json.Unmarshal(data, &response))
	assert.Contains(t, response["spec"], "connectionSettingsHash")
	assert.NotContains(t, response["status"]["connectionSettings"], "desiredHash")
	assert.Contains(t, response["status"]["connectionSettings"], "syncStatus")
	assert.NotContains(t, response["status"]["connectionSettings"], "rotation")
}

func TestMapAgentConnectionSettingsSyncWaitsForNewCertificate(t *testing.T) {
	t.Parallel()

	agent := agentmodel.NewAgent(uuid.New())
	certDER := []byte("new-cert")
	require.NoError(t, agent.ApplyConnectionSettings(&agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: "wss://example.test/api/v1/opamp",
		Certificate: &agentmodel.AgentCertificate{
			Cert: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
		},
	}, nil, nil, nil, nil))
	agent.Status.ConnectionSettingsStatus.Status = agentmodel.ConnectionSettingsStatusApplied
	agent.Status.ConnectionSettingsStatus.LastConnectionSettingsHash = agent.Spec.ConnectionInfo.Hash.Bytes()
	mapper := helper.NewMapper(clock.RealClock{}, 0)
	assert.Equal(t, "pending", mapper.MapAgentToAPI(agent).Status.ConnectionSettings.SyncStatus)

	fingerprint := sha256.Sum256(certDER)
	agent.Status.ActiveClientCertificateHash = fingerprint[:]
	assert.Equal(t, "applied", mapper.MapAgentToAPI(agent).Status.ConnectionSettings.SyncStatus)
}

func TestMapAPIToAgentPackage(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	mapper := helper.NewMapper(clock.RealClock{}, 0)

	//exhaustruct:ignore
	apiPkg := &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       "otelcol",
			Namespace:  "default",
			Attributes: v1.Attributes{"team": "obs"},
			CreatedAt:  v1.NewTime(createdAt),
		},
		Spec: v1.AgentPackageSpec{
			PackageType: "top",
			Version:     "1.2.3",
			DownloadURL: "https://example.com/pkg.tar.gz",
			Headers:     map[string]string{"Authorization": "Bearer x"},
		},
	}

	got := mapper.MapAPIToAgentPackage(apiPkg)

	assert.Equal(t, "otelcol", got.Metadata.Name)
	assert.Equal(t, "default", got.Metadata.Namespace)
	assert.Equal(t, createdAt, got.Metadata.CreatedAt)
	assert.Equal(t, "1.2.3", got.Spec.Version)
	// A client-supplied model never carries the server-managed version.
	assert.Equal(t, int64(0), got.Metadata.ResourceVersion)
}

func TestMapAPIToEndpoint(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	mapper := helper.NewMapper(clock.RealClock{}, 0)

	//exhaustruct:ignore
	apiEndpoint := &v1.Endpoint{
		Metadata: v1.EndpointMetadata{
			Name:       "tempo",
			Namespace:  "default",
			Attributes: v1.Attributes{"team": "obs"},
			CreatedAt:  v1.NewTime(createdAt),
		},
		Spec: v1.EndpointSpec{
			URL:      "https://tempo.example.com",
			Protocol: "otlp",
			Signals:  v1.EndpointSignals{Metrics: false, Logs: false, Traces: true},
			Tenants: []v1.EndpointTenant{
				{
					Name:    "team-a",
					Headers: map[string]string{"X-Scope-OrgID": "team-a"},
					Signals: &v1.EndpointSignals{Metrics: true, Logs: false, Traces: false},
				},
			},
			MetricsQuery: &v1.EndpointMetricsQuery{Metrics: "sum(rate(m[5m]))"},
		},
	}

	got := mapper.MapAPIToEndpoint(apiEndpoint)

	require.NotNil(t, got)
	assert.Equal(t, "tempo", got.Metadata.Name)
	assert.Equal(t, "https://tempo.example.com", got.Spec.URL)
	assert.True(t, got.Spec.Signals.Traces)
	require.Len(t, got.Spec.Tenants, 1)
	require.NotNil(t, got.Spec.Tenants[0].Signals)
	assert.True(t, got.Spec.Tenants[0].Signals.Metrics)
	require.NotNil(t, got.Spec.MetricsQuery)
	assert.Equal(t, "sum(rate(m[5m]))", got.Spec.MetricsQuery.Metrics)
	// A client-supplied model never carries the server-managed version.
	assert.Equal(t, int64(0), got.Metadata.ResourceVersion)
}

func TestMapAPIToEndpoint_Nil(t *testing.T) {
	t.Parallel()

	mapper := helper.NewMapper(clock.RealClock{}, 0)
	assert.Nil(t, mapper.MapAPIToEndpoint(nil))
}
