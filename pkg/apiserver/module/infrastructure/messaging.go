package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	cenats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	"github.com/nats-io/nats.go"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/inmemory"
	natsadapter "github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/nats"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

const (
	// DefaultSenderCloseTimeout is the default timeout for closing the NATS sender.
	DefaultSenderCloseTimeout = 10 * time.Second

	// DefaultReceiverCloseTimeout is the default timeout for closing the NATS receiver.
	DefaultReceiverCloseTimeout = 10 * time.Second

	// EventSubjectSuffix is the suffix for event subjects.
	EventSubjectSuffix = "events"
)

// UnsupportedEventProtocolError is returned when an unsupported event protocol type is specified.
type UnsupportedEventProtocolError struct {
	ProtocolType string
}

// Error implements the error interface.
func (e *UnsupportedEventProtocolError) Error() string {
	return "unsupported event protocol type: " + e.ProtocolType
}

// NewEventSenderAdapter creates the appropriate event sender adapter based on configuration.
// Returns inmemory adapter for standalone mode, or NATS adapter for distributed mode.
//
//nolint:ireturn // Factory function that returns different implementations based on config.
func NewEventSenderAdapter(
	settings *config.EventSettings,
	lifecycle fx.Lifecycle,
) (port.ServerEventSenderPort, error) {
	switch settings.ProtocolType {
	case config.EventProtocolTypeNATS:
		// Create NATS sender
		sender, err := createNATSSender(settings, lifecycle)
		if err != nil {
			return nil, fmt.Errorf("failed to create NATS sender: %w", err)
		}

		// Create NATS receiver
		receiver, err := createNATSReceiver(settings, lifecycle)
		if err != nil {
			return nil, fmt.Errorf("failed to create NATS receiver: %w", err)
		}

		// Use NATS adapter when events are enabled
		return natsadapter.NewEventSenderAdapter(sender, receiver), nil
	case config.EventProtocolTypeInMemory:
		// Unknown protocol type, fall back to in-memory adapter
		return inmemory.NewEventHubAdapter(), nil
	default:
		return nil, &UnsupportedEventProtocolError{ProtocolType: settings.ProtocolType.String()}
	}
}

// NewEventReceiverAdapter creates the appropriate event receiver adapter based on configuration.
// Returns inmemory adapter for standalone mode, or NATS adapter for distributed mode.
//
//nolint:ireturn // Factory function that returns different implementations based on config.
func NewEventReceiverAdapter(
	settings *config.EventSettings,
	lifecycle fx.Lifecycle,
) (port.ServerEventReceiverPort, error) {
	switch settings.ProtocolType {
	case config.EventProtocolTypeNATS:
		// Create NATS sender
		sender, err := createNATSSender(settings, lifecycle)
		if err != nil {
			return nil, fmt.Errorf("failed to create NATS sender: %w", err)
		}

		// Create NATS receiver
		receiver, err := createNATSReceiver(settings, lifecycle)
		if err != nil {
			return nil, fmt.Errorf("failed to create NATS receiver: %w", err)
		}

		// Use NATS adapter when events are enabled
		return natsadapter.NewEventSenderAdapter(sender, receiver), nil
	case config.EventProtocolTypeInMemory:
		// Use in-memory adapter when events are disabled
		return inmemory.NewEventHubAdapter(), nil
	default:
		return nil, &UnsupportedEventProtocolError{ProtocolType: settings.ProtocolType.String()}
	}
}

// createNATSSender creates a NATS sender with lifecycle management.
func createNATSSender(
	settings *config.EventSettings,
	lifecycle fx.Lifecycle,
) (*cenats.Sender, error) {
	endpoint := settings.NATS.Endpoint
	subject := createEventSubject(settings.NATS.SubjectPrefix)
	opts := []nats.Option{}

	sender, err := cenats.NewSender(endpoint, subject, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS sender: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, DefaultSenderCloseTimeout)
			defer cancel()

			err := sender.Close(ctx)
			if err != nil {
				return fmt.Errorf("failed to close NATS sender: %w", err)
			}

			return nil
		},
	})

	return sender, nil
}

// createNATSReceiver creates a NATS receiver with lifecycle management.
func createNATSReceiver(
	settings *config.EventSettings,
	lifecycle fx.Lifecycle,
) (*cenats.Consumer, error) {
	endpoint := settings.NATS.Endpoint
	subject := createEventSubject(settings.NATS.SubjectPrefix)
	opts := []nats.Option{}

	consumer, err := cenats.NewConsumer(endpoint, subject, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS receiver: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return consumer.OpenInbound(ctx)
		},
		OnStop: func(ctx context.Context) error {
			err := consumer.Close(ctx)
			if err != nil {
				return fmt.Errorf("failed to close NATS receiver: %w", err)
			}

			return nil
		},
	})

	return consumer, nil
}

func createEventSubject(subjectPrefix string) string {
	if strings.HasSuffix(subjectPrefix, ".") {
		return subjectPrefix + EventSubjectSuffix
	}

	return subjectPrefix + "." + EventSubjectSuffix
}
