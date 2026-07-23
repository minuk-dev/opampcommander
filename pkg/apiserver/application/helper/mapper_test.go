package helper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
)

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
