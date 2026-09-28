package resourceutil_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/set/internal/resourceutil"
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

			name, value, err := resourceutil.Parse(tt.args, tt.kind)
			if tt.wantErr {
				require.ErrorIs(t, err, resourceutil.ErrInvalidArguments)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantVal, value)
		})
	}
}
