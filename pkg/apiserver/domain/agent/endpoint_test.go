package agentmodel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

func TestNewEndpoint(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	e := agentmodel.NewEndpoint("default", "tempo", agentmodel.Attributes{"team": "obs"}, now, "tester")

	assert.Equal(t, "tempo", e.Metadata.Name)
	assert.Equal(t, "default", e.Metadata.Namespace)
	assert.False(t, e.IsDeleted())

	var created bool

	for _, c := range e.Status.Conditions {
		if c.Type == model.ConditionTypeCreated && c.Status == model.ConditionStatusTrue {
			created = true
		}
	}

	assert.True(t, created, "expected a created condition")
}

func TestEndpointMarkDeleted(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	e := agentmodel.NewEndpoint("default", "tempo", nil, now, "tester")

	assert.False(t, e.IsDeleted())

	deletedAt := now.Add(time.Hour)
	e.MarkDeleted(deletedAt, "remover")

	assert.True(t, e.IsDeleted())
	assert.Equal(t, deletedAt, *e.Metadata.DeletedAt)

	var found bool

	for _, c := range e.Status.Conditions {
		if c.Type == model.ConditionTypeDeleted && c.Status == model.ConditionStatusTrue {
			found = true
		}
	}

	assert.True(t, found, "expected a deleted condition")
}

func TestEndpointEffectiveSignals(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	e := agentmodel.NewEndpoint("default", "mimir", nil, now, "tester")
	e.Spec.Signals = agentmodel.EndpointSignals{Metrics: true, Logs: false, Traces: false}
	e.Spec.Tenants = []agentmodel.EndpointTenant{
		{
			Name:    "team-a",
			Headers: map[string]string{"X-Scope-OrgID": "team-a"},
			Tags:    map[string]string{"tier": "gold"},
			Signals: &agentmodel.EndpointSignals{Metrics: true, Logs: true, Traces: false},
		},
		{
			Name:    "team-b",
			Headers: map[string]string{"X-Scope-OrgID": "team-b"},
			Signals: nil,
		},
	}

	t.Run("tenant override applies", func(t *testing.T) {
		t.Parallel()

		got := e.EffectiveSignals("team-a")
		assert.True(t, got.Logs)
	})

	t.Run("tenant without override inherits endpoint signals", func(t *testing.T) {
		t.Parallel()

		got := e.EffectiveSignals("team-b")
		assert.False(t, got.Logs)
		assert.True(t, got.Metrics)
	})

	t.Run("empty tenant yields endpoint signals", func(t *testing.T) {
		t.Parallel()

		got := e.EffectiveSignals("")
		assert.Equal(t, e.Spec.Signals, got)
	})

	t.Run("unknown tenant yields endpoint signals", func(t *testing.T) {
		t.Parallel()

		got := e.EffectiveSignals("nope")
		assert.Equal(t, e.Spec.Signals, got)
	})
}

func TestEndpointTenantLookup(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	e := agentmodel.NewEndpoint("default", "loki", nil, now, "tester")
	e.Spec.Tenants = []agentmodel.EndpointTenant{{Name: "team-a"}}

	assert.NotNil(t, e.Tenant("team-a"))
	assert.Nil(t, e.Tenant("missing"))
}
