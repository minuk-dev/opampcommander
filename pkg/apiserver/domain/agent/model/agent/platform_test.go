package agent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model/agent"
)

func desc(nonIdentifying map[string]string) agent.Description {
	return agent.Description{
		IdentifyingAttributes:    map[string]string{},
		NonIdentifyingAttributes: nonIdentifying,
	}
}

func TestDescriptionEnvironmentKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		attr map[string]string
		want agent.EnvironmentKind
	}{
		{
			name: "bare host",
			attr: map[string]string{"host.name": "node-1", "host.id": "abc"},
			want: agent.EnvironmentKindHost,
		},
		{
			name: "docker container",
			attr: map[string]string{"container.id": "c123", "container.runtime": "docker"},
			want: agent.EnvironmentKindContainer,
		},
		{
			name: "kubernetes pod without container attrs",
			attr: map[string]string{"k8s.pod.name": "p", "k8s.pod.uid": "u"},
			want: agent.EnvironmentKindContainer,
		},
		{
			name: "container wins over host attrs (k8s node also a host)",
			attr: map[string]string{"host.name": "node-1", "k8s.pod.uid": "u"},
			want: agent.EnvironmentKindContainer,
		},
		{
			name: "nothing reported",
			attr: map[string]string{},
			want: agent.EnvironmentKindUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := desc(tt.attr)
			assert.Equal(t, tt.want, d.EnvironmentKind())
		})
	}
}

func TestDescriptionPlatform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		attr map[string]string
		want agent.Platform
	}{
		{
			name: "bare metal: host, no cloud",
			attr: map[string]string{"host.name": "node-1"},
			want: agent.PlatformBareMetal,
		},
		{
			name: "vm: host with cloud provider",
			attr: map[string]string{"host.name": "i-1", "cloud.provider": "aws"},
			want: agent.PlatformVM,
		},
		{
			name: "docker: container, no k8s",
			attr: map[string]string{"container.id": "c123"},
			want: agent.PlatformDocker,
		},
		{
			name: "kubernetes: k8s attrs present",
			attr: map[string]string{"container.id": "c1", "k8s.pod.uid": "u"},
			want: agent.PlatformKubernetes,
		},
		{
			name: "ecs: cloud.platform aws_ecs",
			attr: map[string]string{"container.id": "c1", "cloud.platform": "aws_ecs"},
			want: agent.PlatformECS,
		},
		{
			name: "kubernetes beats ecs when both k8s and cloud.platform present",
			attr: map[string]string{"k8s.pod.uid": "u", "cloud.platform": "aws_ecs"},
			want: agent.PlatformKubernetes,
		},
		{
			name: "unknown when nothing reported",
			attr: map[string]string{},
			want: agent.PlatformUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := desc(tt.attr)
			assert.Equal(t, tt.want, d.Platform())
		})
	}
}

func TestDescriptionDescriptors(t *testing.T) {
	t.Parallel()

	d := desc(map[string]string{
		"host.id":              "h-1",
		"host.name":            "node-1",
		"host.arch":            "arm64",
		"host.type":            "m5.large",
		"container.id":         "c-1",
		"container.name":       "otelcol",
		"container.image.name": "otel/collector",
		"container.runtime":    "containerd",
		"k8s.pod.uid":          "pod-uid",
		"k8s.node.name":        "node-1",
		"cloud.provider":       "aws",
		"cloud.platform":       "aws_eks",
	})

	assert.Equal(t, agent.Host{ID: "h-1", Name: "node-1", Arch: "arm64", Type: "m5.large"}, d.Host())
	assert.Equal(t, "containerd", d.Container().Runtime)
	assert.Equal(t, "pod-uid", d.K8s().PodUID)
	assert.Equal(t, "aws", d.Cloud().Provider)
	assert.False(t, d.Host().IsZero())
	assert.True(t, agent.Host{}.IsZero())
}
