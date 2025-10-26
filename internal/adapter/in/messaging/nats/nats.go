// Package nats implements NATS messaging adapters.
package nats

import (
	"context"
	"fmt"

	cenats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

const (
	// CloudEventMessageSpec is the CloudEvents spec version used.
	CloudEventMessageSpec = "1.0"
	// CloudEventContentType is the CloudEvents content type used.
	CloudEventContentType = "application/json"
)

var (
	_ port.ServerEventSenderPort = (*EventSenderAdapter)(nil)
)

type EventSenderAdapter struct {
	sender *cenats.Sender
	clock  clock.Clock
}

func NewEventSenderAdapter(
	sender *cenats.Sender,
) *EventSenderAdapter {
	return &EventSenderAdapter{
		sender: sender,
		clock:  clock.NewRealClock(),
	}
}

// SendMessageToServer implements port.ServerEventSenderPort.
func (e *EventSenderAdapter) SendMessageToServer(
	ctx context.Context,
	serverID string,
	message port.ServerMessage,
) error {
	event := cloudevents.NewEvent()

	// Is it better to add message.EventID field?
	eventID := uuid.New().String()
	event.SetID(eventID)
	event.SetSource(newSource(serverID))
	event.SetType(eventTypeFromMessageType(message.Type))
	event.SetSpecVersion(CloudEventMessageSpec)
	event.SetTime(e.clock.Now())

	err := event.SetData(CloudEventContentType, message.ServerMessageForServerToAgent)
	if err != nil {
		return fmt.Errorf("failed to set event data for server %s: %w", serverID, err)
	}

	err = e.sender.Send(ctx, binding.ToMessage(&event))
	if err != nil {
		return fmt.Errorf("failed to send message to server %s: %w", serverID, err)
	}

	return nil
}

func newSource(serverID string) string {
	return "opampcommander/server/" + serverID
}

func eventTypeFromMessageType(messageType port.ServerMessageType) string {
	switch messageType {
	case port.ServerMessageTypeSendServerToAgent:
		return "io.opampcommander.server.sendtosagent.v1"
	default:
		return "io.opampcommander.server.unknown.v1"
	}
}
