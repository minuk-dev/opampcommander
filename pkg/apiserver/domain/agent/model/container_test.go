package agentmodel_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model/agent"
)

func TestContainerIDOf(t *testing.T) {
	t.Parallel()

	t.Run("prefers k8s.pod.uid", func(t *testing.T) {
		t.Parallel()

		d := agent.Description{NonIdentifyingAttributes: map[string]string{"k8s.pod.uid": "u", "container.id": "c"}}
		assert.Equal(t, "u", agentmodel.ContainerIDOf(d))
	})

	t.Run("falls back to container.id", func(t *testing.T) {
		t.Parallel()

		d := agent.Description{NonIdentifyingAttributes: map[string]string{"container.id": "c"}}
		assert.Equal(t, "c", agentmodel.ContainerIDOf(d))
	})

	t.Run("empty when no container attrs", func(t *testing.T) {
		t.Parallel()

		d := agent.Description{NonIdentifyingAttributes: map[string]string{"host.name": "n"}}
		assert.Empty(t, agentmodel.ContainerIDOf(d))
	})
}

func TestContainerObserveAgent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	uid := uuid.New()

	t.Run("kubernetes pod links to node host", func(t *testing.T) {
		t.Parallel()

		container := agentmodel.NewContainer("pod-uid", now)
		desc := agent.Description{NonIdentifyingAttributes: map[string]string{
			"container.image.name": "otel/collector",
			"container.runtime":    "containerd",
			"k8s.pod.uid":          "pod-uid",
			"k8s.pod.name":         "otelcol-abc",
			"k8s.node.name":        "node-1",
		}}

		container.ObserveAgent(uid, desc, now)

		assert.Equal(t, "otelcol-abc", container.Metadata.Name)
		assert.Equal(t, agent.PlatformKubernetes, container.Spec.Platform)
		assert.Equal(t, "otel/collector", container.Spec.ImageName)
		// No host.* reported, so the node name links the container to its host.
		assert.Equal(t, "node-1", container.Spec.HostID)
		assert.Equal(t, []uuid.UUID{uid}, container.Status.AgentInstanceUIDs)
	})

	t.Run("docker container prefers reported host id", func(t *testing.T) {
		t.Parallel()

		container := agentmodel.NewContainer("c-1", now)
		desc := agent.Description{NonIdentifyingAttributes: map[string]string{
			"container.id":   "c-1",
			"container.name": "otelcol",
			"host.id":        "h-9",
		}}

		container.ObserveAgent(uid, desc, now)

		assert.Equal(t, "otelcol", container.Metadata.Name)
		assert.Equal(t, agent.PlatformDocker, container.Spec.Platform)
		assert.Equal(t, "h-9", container.Spec.HostID)
	})
}
