package application

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
)

func TestFormatAgentsShort(t *testing.T) {
	instanceUID := uuid.New()
	var output bytes.Buffer

	err := formatAgents(&output, []v1.Agent{{
		Metadata: v1.AgentMetadata{
			InstanceUID: instanceUID,
			Namespace:   "default",
		},
	}}, formatter.SHORT)
	if err != nil {
		t.Fatalf("formatAgents() error = %v", err)
	}

	for _, want := range []string{"Instance UID", instanceUID.String()} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("formatAgents() output = %q, want %q", output.String(), want)
		}
	}
}
