// Package kafka implements Kafka messaging adapters.
package kafka

import (
	"context"
	"fmt"

	observabilityClient "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/client"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/google/uuid"

	kafkamodel "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/kafka"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var (
	_ agentport.ServerEventSenderPort = (*EventSenderAdapter)(nil)
)

// EventSenderAdapter implements agentport.ServerEventSenderPort using Kafka CloudEvents sender.
type EventSenderAdapter struct {
	sender cloudevents.Client
	clock  clock.Clock
}

// NewEventSenderAdapter creates a new EventSenderAdapter.
func NewEventSenderAdapter(
	protocolSender protocol.Sender,
) (*EventSenderAdapter, error) {
	//nolint:godox
	// TODO: cloudevents's observability does not support to inject TracerProvider instead of global
	// https://github.com/cloudevents/sdk-go/pull/1202
	otelService := observabilityClient.NewOTelObservabilityService()

	sender, err := cloudevents.NewClient(protocolSender, client.WithObservabilityService(otelService))
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudEvents client for sender: %w", err)
	}

	return &EventSenderAdapter{
		sender: sender,
		clock:  clock.NewRealClock(),
	}, nil
}

// SendMessageToServer implements agentport.ServerEventSenderPort.
func (e *EventSenderAdapter) SendMessageToServer(
	ctx context.Context,
	server *agentmodel.Server,
	message serverevent.Message,
) error {
	serverID := server.ID
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
