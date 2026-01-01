package model_test

import (
	"testing"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

func TestRemoteConfigStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    model.RemoteConfigStatus
		expected protobufs.RemoteConfigStatuses
	}{
		{
			name:     "NotSet",
			input:    model.RemoteConfigStatusUnset,
			expected: protobufs.RemoteConfigStatuses_RemoteConfigStatuses_UNSET,
		},
		{
			name:     "Applied",
			input:    model.RemoteConfigStatusApplied,
			expected: protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
		},
		{
			name:     "Applying",
			input:    model.RemoteConfigStatusApplying,
			expected: protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLYING,
		},
		{
			name:     "Failed",
			input:    model.RemoteConfigStatusFailed,
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
