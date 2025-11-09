// Package nats implements NATS messaging adapters.
package nats

import (
	"context"
	"fmt"
	"log/slog"

	observabilityClient "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cenats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/client"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

const (
	// CloudEventMessageSpec is the CloudEvents spec version used.
	CloudEventMessageSpec = "1.0"
	// CloudEventContentType is the CloudEvents content type used.
	CloudEventContentType = "application/json"

	sendToAgentEventType = "io.opampcommander.server.sendtosagent.v1"
	unknownEventType     = "io.opampcommander.server.unknown.v1"
)

var (
	_ port.ServerEventSenderPort   = (*EventSenderAdapter)(nil)
	_ port.ServerEventReceiverPort = (*EventSenderAdapter)(nil)
)

// EventSenderAdapter implements port.ServerEventSenderPort using NATS CloudEvents sender.
type EventSenderAdapter struct {
	sender   cloudevents.Client
	receiver cloudevents.Client
	logger   *slog.Logger
	clock    clock.Clock
}

// NewEventSenderAdapter creates a new EventSenderAdapter.
func NewEventSenderAdapter(
	natsSender *cenats.Sender,
	natsReceiver *cenats.Consumer,
	logger *slog.Logger,
) (*EventSenderAdapter, error) {
	otelService := observabilityClient.NewOTelObservabilityService()

	var opts []client.Option

	opts = append(opts, client.WithObservabilityService(otelService))

	sender, err := cloudevents.NewClient(natsSender, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudEvents client for sender: %w", err)
	}

	receiver, err := cloudevents.NewClient(natsReceiver, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudEvents client for receiver: %w", err)
	}

	// sender can be nil when events are disabled
	return &EventSenderAdapter{
		sender:   sender,
		receiver: receiver,
		logger:   logger,
		clock:    clock.NewRealClock(),
	}, nil
}

// SendMessageToServer implements port.ServerEventSenderPort.
func (e *EventSenderAdapter) SendMessageToServer(
	ctx context.Context,
	serverID string,
	message serverevent.Message,
) error {
	event := cloudevents.NewEvent()

	// Is it better to add message.EventID field?
	eventID := uuid.New().String()
	event.SetID(eventID)
	event.SetSource(newSource(serverID))
	event.SetType(eventTypeFromMessageType(message.Type))
	event.SetSpecVersion(CloudEventMessageSpec)
	event.SetTime(e.clock.Now())

	err := event.SetData(CloudEventContentType, message.Payload)
	if err != nil {
		return fmt.Errorf("failed to set event data for server %s: %w", serverID, err)
	}

	err = e.sender.Send(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to send message to server %s: %w", serverID, err)
	}

	return nil
}

// StartReceiver implements port.ServerEventReceiverPort.
func (e *EventSenderAdapter) StartReceiver(ctx context.Context, handler port.ReceiveServerEventHandler) error {
	err := e.receiver.StartReceiver(ctx, func(ctx context.Context, event event.Event) {
		var payload serverevent.MessagePayload

		err := event.DataAs(&payload)
		if err != nil {
			e.logger.Warn("failed to parse event data",
				slog.String("eventID", event.ID()),
				slog.String("eventType", event.Type()),
				slog.String("error", err.Error()),
			)

			return
		}

		messageType, err := messageTypeFromEventType(event.Type())
		if err != nil {
			e.logger.Warn("unknown event type",
				slog.String("eventID", event.ID()),
				slog.String("eventType", event.Type()),
				slog.String("error", err.Error()),
			)

			return
		}

		message := &serverevent.Message{
			Source:  event.Source(),
			Target:  "",
			Type:    messageType,
			Payload: payload,
		}

		err = handler(ctx, message)
		if err != nil {
			e.logger.Warn("failed to handle received message",
				slog.String("eventID", event.ID()),
				slog.String("eventType", event.Type()),
				slog.String("error", err.Error()),
			)

			return
		}
	})
	if err != nil {
		return fmt.Errorf("failed to start receiver: %w", err)
	}

	return nil
}

func newSource(serverID string) string {
	return "opampcommander/server/" + serverID
}

func eventTypeFromMessageType(messageType serverevent.MessageType) string {
	switch messageType {
	case serverevent.MessageTypeSendServerToAgent:
		return sendToAgentEventType
	default:
		return unknownEventType
	}
}

func messageTypeFromEventType(eventType string) (serverevent.MessageType, error) {
	switch eventType {
	case sendToAgentEventType:
		return serverevent.MessageTypeSendServerToAgent, nil
	default:
		return "", &UnknownMessageTypeError{MessageType: eventType}
	}
}
