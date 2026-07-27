package direct_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commondirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/direct"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	t.Parallel()

	agentUID := uuid.New()
	original := serverevent.Message{
		Source: "server-a",
		Target: "server-b",
		Type:   serverevent.MessageTypeSendServerToAgent,
		Payload: serverevent.MessagePayload{
			MessageForServerToAgent: &serverevent.MessageForServerToAgent{
				TargetAgentInstanceUIDs: []uuid.UUID{agentUID},
			},
			MessageForInvalidateAgentCache: nil,
		},
	}

	got, err := commondirect.ToEnvelope(original).ToMessage()
	require.NoError(t, err)

	assert.Equal(t, original.Source, got.Source)
	assert.Equal(t, original.Target, got.Target)
	assert.Equal(t, original.Type, got.Type)
	require.NotNil(t, got.Payload.MessageForServerToAgent)
	assert.Equal(t, []uuid.UUID{agentUID}, got.Payload.TargetAgentInstanceUIDs)
}

func TestParseMessageType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		raw       string
		expected  serverevent.MessageType
		expectErr bool
	}{
		{"send", "SendServerToAgent", serverevent.MessageTypeSendServerToAgent, false},
		{"invalidate", "InvalidateAgentCache", serverevent.MessageTypeInvalidateAgentCache, false},
		{"unknown", "Bogus", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := commondirect.ParseMessageType(tt.raw)
			if tt.expectErr {
				require.ErrorIs(t, err, commondirect.ErrUnknownMessageType)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestPayloadEncodeDecode(t *testing.T) {
	t.Parallel()

	agentUID := uuid.New()
	payload := serverevent.MessagePayload{
		MessageForServerToAgent: nil,
		MessageForInvalidateAgentCache: &serverevent.MessageForInvalidateAgentCache{
			AgentInstanceUIDs: []uuid.UUID{agentUID},
		},
	}

	data, err := commondirect.EncodePayload(payload)
	require.NoError(t, err)

	got, err := commondirect.DecodePayload(data)
	require.NoError(t, err)

	require.NotNil(t, got.MessageForInvalidateAgentCache)
	assert.Equal(t, []uuid.UUID{agentUID}, got.AgentInstanceUIDs)
}
