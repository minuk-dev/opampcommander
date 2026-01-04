// Package kafka provides Kafka messaging adapter implementations.
package kafka

import (
	"context"
	"fmt"
	"log/slog"

	observabilityClient "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/client"
	"github.com/cloudevents/sdk-go/v2/event"

	kafkamodel "github.com/minuk-dev/opampcommander/internal/adapter/common/kafka"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var (
	_ port.ServerEventReceiverPort = (*EventReceiverAdapter)(nil)
)

// EventReceiverAdapter implements port.ServerEventReceiverPort using Kafka CloudEvents receiver.
type EventReceiverAdapter struct {
	serverIdentityProvider port.ServerIdentityProvider
	receiver               cloudevents.Client
	logger                 *slog.Logger
	clock                  clock.Clock
}

// NewEventReceiverAdapter creates a new EventReceiverAdapter.
func NewEventReceiverAdapter(
	serverIdentityProvider port.ServerIdentityProvider,
	protocolConsumer *cekafka.Consumer,
	logger *slog.Logger,
) (*EventReceiverAdapter, error) {
	//nolint:godox
	// TODO: cloudevents's observability does not support to inject TracerProvider instead of global
	// https://github.com/cloudevents/sdk-go/pull/1202
	otelService := observabilityClient.NewOTelObservabilityService()

	var opts []client.Option

	opts = append(opts, client.WithObservabilityService(otelService))

	receiver, err := cloudevents.NewClient(protocolConsumer, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudEvents client for receiver: %w", err)
	}

	return &EventReceiverAdapter{
		serverIdentityProvider: serverIdentityProvider,
		receiver:               receiver,
		logger:                 logger,
		clock:                  clock.NewRealClock(),
	}, nil
}

// StartReceiver implements port.ServerEventReceiverPort.
func (e *EventReceiverAdapter) StartReceiver(
	ctx context.Context,
	handler port.ReceiveServerEventHandler,
) error {
	err := e.receiver.StartReceiver(ctx, func(ctx context.Context, event event.Event) {
		logArgs := []any{
			slog.String("eventID", event.ID()),
			slog.String("eventType", event.Type()),
		}
		if !e.isRelatedEvent(ctx, event) {
			return
		}

		message, err := eventToMessage(event)
		if err != nil {
			e.logger.Warn("failed to convert event to message",
				append(logArgs, slog.String("error", err.Error()))...,
			)

			return
		}

		err = handler(ctx, message)
		if err != nil {
			e.logger.Warn("failed to convert event to message",
				append(logArgs, slog.String("error", err.Error()))...,
			)

			return
		}
	})
	if err != nil {
		return fmt.Errorf("failed to start receiver: %w", err)
	}

	return nil
}

func (e *EventReceiverAdapter) isRelatedEvent(
	ctx context.Context,
	event event.Event,
) bool {
	currentServer, err := e.serverIdentityProvider.CurrentServer(ctx)
	if err != nil {
		e.logger.Warn("failed to get current server identity",
			slog.String("eventID", event.ID()),
			slog.String("eventType", event.Type()),
		)

		return false
	}

	targetServer := event.Subject()
	if targetServer != currentServer.ID {
		// skip events not targeted to this server
		e.logger.Info("skipping event not targeted to this server",
			slog.String("eventID", event.ID()),
			slog.String("eventType", event.Type()),
			slog.String("targetServer", targetServer),
			slog.String("currentServer", currentServer.ID),
		)

		return false
	}

	return true
}

func eventToMessage(event event.Event) (*serverevent.Message, error) {
	var payload serverevent.MessagePayload

	err := event.DataAs(&payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event data: %w", err)
	}

	messageType, err := kafkamodel.MessageTypeFromEventType(event.Type())
	if err != nil {
		return nil, fmt.Errorf("unknown event type: %w", err)
	}

	message := &serverevent.Message{
		Source:  event.Source(),
		Target:  event.Subject(),
		Type:    messageType,
		Payload: payload,
	}

	return message, nil
}
