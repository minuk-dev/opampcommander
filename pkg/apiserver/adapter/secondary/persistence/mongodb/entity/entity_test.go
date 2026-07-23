package entity_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model/vo"
)

func TestAgentPackageEntity_RoundTrip(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	deletedAt := createdAt.Add(time.Hour)
	domainPkg := &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{
			Name:            "otelcol",
			Namespace:       "default",
			Attributes:      agentmodel.Attributes{"team": "obs"},
			ResourceVersion: 7,
			CreatedAt:       createdAt,
			DeletedAt:       &deletedAt,
		},
		Spec: agentmodel.AgentPackageSpec{
			PackageType: "top",
			Version:     "1.2.3",
			DownloadURL: "https://example.com/pkg.tar.gz",
			ContentHash: []byte{0x01, 0x02},
			Signature:   []byte{0x03},
			Headers:     map[string]string{"Authorization": "Bearer x"},
			Hash:        []byte{0x04},
		},
		Status: agentmodel.AgentPackageStatus{
			Conditions: []model.Condition{{Type: model.ConditionTypeCreated, Status: model.ConditionStatusTrue}},
		},
	}

	got := entity.AgentPackageFromDomain(domainPkg).ToDomain()

	assert.Equal(t, int64(7), got.Metadata.ResourceVersion, "ResourceVersion must survive the round trip")
	assert.Equal(t, "otelcol", got.Metadata.Name)
	assert.Equal(t, "default", got.Metadata.Namespace)
	assert.Equal(t, agentmodel.Attributes{"team": "obs"}, got.Metadata.Attributes)
	assert.Equal(t, createdAt, got.Metadata.CreatedAt)
	require.NotNil(t, got.Metadata.DeletedAt)
	assert.Equal(t, deletedAt, *got.Metadata.DeletedAt)
	assert.Equal(t, domainPkg.Spec, got.Spec)
	require.Len(t, got.Status.Conditions, 1)
	assert.Equal(t, model.ConditionTypeCreated, got.Status.Conditions[0].Type)
}

func TestHostEntity_RoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	uid := uuid.New()
	domainHost := &agentmodel.Host{
		Metadata: agentmodel.HostMetadata{
			ID:              "h-1",
			Name:            "node-1",
			Labels:          map[string]string{"env": "prod"},
			Annotations:     map[string]string{"note": "x"},
			ResourceVersion: 4,
			FirstSeenAt:     now,
			LastSeenAt:      now.Add(time.Minute),
		},
		Spec: agentmodel.HostSpec{
			Platform: agent.PlatformVM,
			Arch:     "amd64",
			Type:     "m5.large",
			OS:       vo.OS{Type: "linux", Version: "5.15"},
			Cloud:    agent.Cloud{Provider: "aws", Platform: "aws_ec2", Region: "us-east-1"},
		},
		Status: agentmodel.HostStatus{
			AgentInstanceUIDs: []uuid.UUID{uid},
			Conditions:        []model.Condition{{Type: model.ConditionTypeCreated, Status: model.ConditionStatusTrue}},
		},
	}

	got := entity.HostFromDomain(domainHost).ToDomain()

	assert.Equal(t, int64(4), got.Metadata.ResourceVersion, "ResourceVersion must survive the round trip")
	assert.Equal(t, "h-1", got.Metadata.ID)
	assert.Equal(t, "node-1", got.Metadata.Name)
	assert.Equal(t, domainHost.Spec, got.Spec)
	require.Len(t, got.Status.AgentInstanceUIDs, 1)
	assert.Equal(t, uid, got.Status.AgentInstanceUIDs[0])
}

func TestContainerEntity_RoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	uid := uuid.New()
	domainContainer := &agentmodel.Container{
		Metadata: agentmodel.ContainerMetadata{
			ID:              "pod-uid",
			Name:            "otelcol-0",
			Labels:          map[string]string{"app": "otel"},
			Annotations:     map[string]string{"note": "y"},
			ResourceVersion: 9,
			FirstSeenAt:     now,
			LastSeenAt:      now.Add(time.Minute),
		},
		Spec: agentmodel.ContainerSpec{
			Platform:  agent.PlatformKubernetes,
			ImageName: "otel/opentelemetry-collector",
			Runtime:   "containerd",
			HostID:    "node-1",
			K8s: agent.K8s{
				PodName:       "otelcol-0",
				PodUID:        "pod-uid",
				NamespaceName: "monitoring",
				NodeName:      "node-1",
				ContainerName: "otelcol",
			},
		},
		Status: agentmodel.ContainerStatus{
			AgentInstanceUIDs: []uuid.UUID{uid},
			Conditions:        []model.Condition{{Type: model.ConditionTypeCreated, Status: model.ConditionStatusTrue}},
		},
	}

	got := entity.ContainerFromDomain(domainContainer).ToDomain()

	assert.Equal(t, int64(9), got.Metadata.ResourceVersion, "ResourceVersion must survive the round trip")
	assert.Equal(t, "pod-uid", got.Metadata.ID)
	assert.Equal(t, domainContainer.Spec, got.Spec)
	require.Len(t, got.Status.AgentInstanceUIDs, 1)
	assert.Equal(t, uid, got.Status.AgentInstanceUIDs[0])
}

func TestEndpointEntity_RoundTrip(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	tenantSignals := &agentmodel.EndpointSignals{Metrics: true, Logs: false, Traces: false}
	domainEndpoint := &agentmodel.Endpoint{
		Metadata: agentmodel.EndpointMetadata{
			Name:            "tempo",
			Namespace:       "default",
			Attributes:      agentmodel.Attributes{"team": "obs"},
			ResourceVersion: 3,
			CreatedAt:       createdAt,
			DeletedAt:       nil,
		},
		Spec: agentmodel.EndpointSpec{
			URL:      "https://tempo.example.com",
			Protocol: "otlp",
			Signals:  agentmodel.EndpointSignals{Metrics: false, Logs: false, Traces: true},
			Tenants: []agentmodel.EndpointTenant{
				{
					Name:    "team-a",
					Headers: map[string]string{"X-Scope-OrgID": "team-a"},
					Tags:    map[string]string{"tier": "gold"},
					Signals: tenantSignals,
				},
			},
			MetricsQuery: &agentmodel.EndpointMetricsQuery{
				Metrics: "sum(rate(m[5m]))",
				Logs:    "",
				Traces:  "",
			},
		},
		Status: agentmodel.EndpointStatus{
			Conditions: []model.Condition{{Type: model.ConditionTypeCreated, Status: model.ConditionStatusTrue}},
		},
	}

	got := entity.EndpointResourceEntityFromDomain(domainEndpoint).ToDomain()

	assert.Equal(t, int64(3), got.Metadata.ResourceVersion, "ResourceVersion must survive the round trip")
	assert.Equal(t, "tempo", got.Metadata.Name)
	assert.Equal(t, domainEndpoint.Spec, got.Spec)
	require.Len(t, got.Spec.Tenants, 1)
	require.NotNil(t, got.Spec.Tenants[0].Signals)
	assert.True(t, got.Spec.Tenants[0].Signals.Metrics)
	require.NotNil(t, got.Spec.MetricsQuery)
	assert.Equal(t, "sum(rate(m[5m]))", got.Spec.MetricsQuery.Metrics)
}
