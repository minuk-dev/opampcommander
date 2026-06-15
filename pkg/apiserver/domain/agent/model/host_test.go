package agentmodel_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model/agent"
)

func TestHostIDOf(t *testing.T) {
	t.Parallel()

	t.Run("prefers host.id", func(t *testing.T) {
		t.Parallel()

		d := agent.Description{NonIdentifyingAttributes: map[string]string{"host.id": "h-1", "host.name": "n"}}
		assert.Equal(t, "h-1", agentmodel.HostIDOf(d))
	})

	t.Run("falls back to host.name", func(t *testing.T) {
		t.Parallel()

		d := agent.Description{NonIdentifyingAttributes: map[string]string{"host.name": "n"}}
		assert.Equal(t, "n", agentmodel.HostIDOf(d))
	})

	t.Run("empty when no host attrs", func(t *testing.T) {
		t.Parallel()

		d := agent.Description{NonIdentifyingAttributes: map[string]string{"container.id": "c"}}
		assert.Empty(t, agentmodel.HostIDOf(d))
	})
}

func TestHostObserveAgent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)
	uid1 := uuid.New()
	uid2 := uuid.New()

	host := agentmodel.NewHost("h-1", now)
	desc := agent.Description{NonIdentifyingAttributes: map[string]string{
		"host.id":        "h-1",
		"host.name":      "node-1",
		"host.arch":      "amd64",
		"host.type":      "m5.large",
		"os.type":        "linux",
		"cloud.provider": "aws",
	}}

	host.ObserveAgent(uid1, desc, now)

	assert.Equal(t, "node-1", host.Metadata.Name)
	assert.Equal(t, agent.PlatformVM, host.Spec.Platform)
	assert.Equal(t, "amd64", host.Spec.Arch)
	assert.Equal(t, "linux", host.Spec.OS.Type)
	assert.Equal(t, "aws", host.Spec.Cloud.Provider)
	assert.Equal(t, []uuid.UUID{uid1}, host.Status.AgentInstanceUIDs)
	assert.Equal(t, now, host.Metadata.LastSeenAt)

	// A second agent on the same host advances LastSeenAt and is deduplicated.
	host.ObserveAgent(uid2, desc, later)
	host.ObserveAgent(uid1, desc, later)

	assert.Equal(t, []uuid.UUID{uid1, uid2}, host.Status.AgentInstanceUIDs)
	assert.Equal(t, later, host.Metadata.LastSeenAt)
	assert.Equal(t, now, host.Metadata.FirstSeenAt)
}
