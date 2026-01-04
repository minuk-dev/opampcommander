package kafka_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kafkamodel "github.com/minuk-dev/opampcommander/internal/adapter/common/kafka"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
)

func TestEventTypeFromMessageType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		messageType serverevent.MessageType
		expected    string
	}{
		{
			name:        "SendServerToAgent type",
			messageType: serverevent.MessageTypeSendServerToAgent,
			expected:    kafkamodel.SendToAgentEventType,
		},
		{
			name:        "Unknown type",
			messageType: "unknown",
			expected:    kafkamodel.UnknownEventType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := kafkamodel.EventTypeFromMessageType(tt.messageType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageTypeFromEventType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		eventType   string
		expected    serverevent.MessageType
		expectError bool
	}{
		{
			name:        "SendToAgent event type",
			eventType:   kafkamodel.SendToAgentEventType,
			expected:    serverevent.MessageTypeSendServerToAgent,
			expectError: false,
		},
		{
			name:        "Unknown event type",
			eventType:   "unknown",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := kafkamodel.MessageTypeFromEventType(tt.eventType)

			if tt.expectError {
				require.Error(t, err)

				var unknownErr *kafkamodel.UnknownMessageTypeError
				assert.ErrorAs(t, err, &unknownErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
