package agentmodel_test

import (
	"testing"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
)

func TestRemoteConfigStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    agentmodel.RemoteConfigStatus
		expected protobufs.RemoteConfigStatuses
	}{
		{
			name:     "NotSet",
			input:    agentmodel.RemoteConfigStatusUnset,
			expected: protobufs.RemoteConfigStatuses_RemoteConfigStatuses_UNSET,
		},
		{
			name:     "Applied",
			input:    agentmodel.RemoteConfigStatusApplied,
			expected: protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
		},
		{
			name:     "Applying",
			input:    agentmodel.RemoteConfigStatusApplying,
			expected: protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLYING,
		},
		{
			name:     "Failed",
			input:    agentmodel.RemoteConfigStatusFailed,
			expected: protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, int32(tt.expected), int32(tt.input))
		})
	}
}
