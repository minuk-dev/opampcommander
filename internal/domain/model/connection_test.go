package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

func TestConnectionTypeFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected model.ConnectionType
	}{
		{"WebSocket", model.ConnectionTypeWebSocket},
		{"HTTP", model.ConnectionTypeHTTP},
		{"Unknown", model.ConnectionTypeUnknown},
		{"InvalidType", model.ConnectionTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result := model.ConnectionTypeFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnectionType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    model.ConnectionType
		expected string
	}{
		{model.ConnectionTypeWebSocket, "WebSocket"},
		{model.ConnectionTypeHTTP, "HTTP"},
		{model.ConnectionTypeUnknown, "Unknown"},
		{model.ConnectionType(999), "Unknown"}, // Test for an undefined value
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			result := tt.input.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
