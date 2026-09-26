//nolint:lll // Table cases keep input and expected identity together.
package agentmodel_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
)

func TestApplicationIDOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		desc agent.Description
		want string
	}{
		{"namespace and name", agent.Description{IdentifyingAttributes: map[string]string{"service.namespace": "payments", "service.name": "api"}}, "cGF5bWVudHMAYXBp"},
		{"name without namespace", agent.Description{IdentifyingAttributes: map[string]string{"service.name": "api"}}, "AGFwaQ"},
		{"missing name", agent.Description{IdentifyingAttributes: map[string]string{"service.namespace": "payments"}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, agentmodel.ApplicationIDOf(tt.desc))
		})
	}
}

func TestApplicationObserveAgent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	application := agentmodel.NewApplication("cGF5bWVudHMAcGFp", now)
	first := uuid.New()
	second := uuid.New()

	application.ObserveAgent(first, agent.Description{IdentifyingAttributes: map[string]string{
		"service.namespace": "payments", "service.name": "api", "service.version": "1.0.0",
	}}, now)
	application.ObserveAgent(second, agent.Description{IdentifyingAttributes: map[string]string{
		"service.namespace": "payments", "service.name": "api", "service.version": "1.1.0",
	}}, now.Add(time.Minute))
	application.ObserveAgent(first, agent.Description{IdentifyingAttributes: map[string]string{
		"service.namespace": "payments", "service.name": "api", "service.version": "1.0.0",
	}}, now.Add(2*time.Minute))

	assert.Equal(t, "api", application.Metadata.Name)
	assert.Equal(t, "payments", application.Spec.Namespace)
	assert.Equal(t, []string{"1.0.0", "1.1.0"}, application.Spec.Versions)
	assert.Equal(t, []uuid.UUID{first, second}, application.Status.AgentInstanceUIDs)
	assert.Equal(t, now.Add(2*time.Minute), application.Metadata.LastSeenAt)
}
