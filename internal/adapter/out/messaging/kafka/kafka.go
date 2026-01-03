// Package kafka implements Kafka messaging adapters.
package kafka

import (
	"context"
	"fmt"
	"log/slog"

	observabilityClient "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/client"
	"github.com/google/uuid"

	kafkamodel "github.com/minuk-dev/opampcommander/internal/adapter/common/kafka"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var (
	_ port.ServerEventSenderPort = (*EventSenderAdapter)(nil)
)

// EventSenderAdapter implements port.ServerEventSenderPort using Kafka CloudEvents sender.
type EventSenderAdapter struct {
	sender cloudevents.Client
	logger *slog.Logger
	clock  clock.Clock
}

// NewEventSenderAdapter creates a new EventSenderAdapter.
func NewEventSenderAdapter(
	protocolSender *cekafka.Sender,
	logger *slog.Logger,
) (*EventSenderAdapter, error) {
	//nolint:godox
	// TODO: cloudevents's observability does not support to inject TracerProvider instead of global
	// https://github.com/cloudevents/sdk-go/pull/1202
	otelService := observabilityClient.NewOTelObservabilityService()

	var opts []client.Option

	opts = append(opts, client.WithObservabilityService(otelService))

	sender, err := cloudevents.NewClient(protocolSender, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudEvents client for sender: %w", err)
	}

	// sender can be nil when events are disabled
	return &EventSenderAdapter{
		sender: sender,
		logger: logger,
		clock:  clock.NewRealClock(),
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
	event.SetSubject(message.Target) // use targetServer as subject
	event.SetType(kafkamodel.EventTypeFromMessageType(message.Type))
	event.SetSpecVersion(kafkamodel.CloudEventMessageSpec)
	event.SetTime(e.clock.Now())

	err := event.SetData(kafkamodel.CloudEventContentType, message.Payload)
	if err != nil {
		return fmt.Errorf("failed to set event data for server %s: %w", serverID, err)
	}

	err = e.sender.Send(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to send message to server %s: %w", serverID, err)
	}

	return nil
}

func newSource(serverID string) string {
	return "opampcommander/server/" + serverID
}
