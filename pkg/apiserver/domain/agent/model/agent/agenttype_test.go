package agent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model/agent"
)

func TestDescriptionAgentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		serviceName string
		want        agent.Type
		isCollector bool
	}{
		{
			name:        "otel collector core",
			serviceName: "otelcol",
			want:        agent.TypeOTelCollector,
			isCollector: true,
		},
		{
			name:        "otel collector contrib",
			serviceName: "otelcol-contrib",
			want:        agent.TypeOTelCollectorContrib,
			isCollector: true,
		},
		{
			name:        "otel collector k8s",
			serviceName: "otelcol-k8s",
			want:        agent.TypeOTelCollectorK8s,
			isCollector: true,
		},
		{
			name:        "unknown distribution is preserved verbatim",
			serviceName: "otelcol-mydistro",
			want:        agent.Type("otelcol-mydistro"),
			isCollector: true,
		},
		{
			name:        "custom-branded collector is not recognized",
			serviceName: "splunk-otel-collector",
			want:        agent.TypeUnknown,
			isCollector: false,
		},
		{
			name:        "non collector service",
			serviceName: "my-app",
			want:        agent.TypeUnknown,
			isCollector: false,
		},
		{
			name:        "empty service name",
			serviceName: "",
			want:        agent.TypeUnknown,
			isCollector: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := agent.Description{
				IdentifyingAttributes:    map[string]string{"service.name": tt.serviceName},
				NonIdentifyingAttributes: map[string]string{},
			}

			got := d.AgentType()
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.isCollector, got.IsOTelCollector())
		})
	}
}
