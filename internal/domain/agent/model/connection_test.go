package agentmodel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
)

func TestConnectionTypeFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected agentmodel.ConnectionType
	}{
		{"WebSocket", agentmodel.ConnectionTypeWebSocket},
		{"HTTP", agentmodel.ConnectionTypeHTTP},
		{"Unknown", agentmodel.ConnectionTypeUnknown},
		{"InvalidType", agentmodel.ConnectionTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result := agentmodel.ConnectionTypeFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnectionType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    agentmodel.ConnectionType
		expected string
	}{
		{agentmodel.ConnectionTypeWebSocket, "WebSocket"},
		{agentmodel.ConnectionTypeHTTP, "HTTP"},
		{agentmodel.ConnectionTypeUnknown, "Unknown"},
		{agentmodel.ConnectionType(999), "Unknown"}, // Test for an undefined value
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			result := tt.input.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
