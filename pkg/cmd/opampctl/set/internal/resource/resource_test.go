package resource_test

import (
	"testing"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/set/internal/resource"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		kind     string
		wantName string
		wantVal  string
		wantErr  bool
	}{
		{name: "slash resource", args: []string{"agent/old", "new"}, kind: "agent", wantName: "old", wantVal: "new"},
		{
			name: "split resource", args: []string{"agentgroup", "group", "cert"},
			kind: "agentgroup", wantName: "group", wantVal: "cert",
		},
		{name: "old resource-first command", args: []string{"agent", "old"}, kind: "agent", wantErr: true},
		{name: "wrong resource", args: []string{"agentgroup/group", "cert"}, kind: "agent", wantErr: true},
		{name: "missing name", args: []string{"agent/", "new"}, kind: "agent", wantErr: true},
		{name: "missing value", args: []string{"agent/old", ""}, kind: "agent", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			name, value, err := resource.Parse(tt.args, tt.kind)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q, %q) error = %v, wantErr %t", tt.args, tt.kind, err, tt.wantErr)
			}

			if name != tt.wantName || value != tt.wantVal {
				t.Errorf("Parse(%q, %q) = (%q, %q), want (%q, %q)",
					tt.args, tt.kind, name, value, tt.wantName, tt.wantVal)
			}
		})
	}
}
