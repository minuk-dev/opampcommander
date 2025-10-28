package event

import (
	"context"
	"fmt"
	"strings"
	"time"

	cenats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	"github.com/nats-io/nats.go"
	"go.uber.org/fx"

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

// NewEventSenderOptional creates a new NATS event sender only when events are enabled.
func NewEventSenderOptional(
	settings *config.EventSettings,
	lifecycle fx.Lifecycle,
) (*cenats.Sender, error) {
	if !settings.Enabled {
		// Return nil when events are disabled
		// This is acceptable in fx.Provide context
		return nil, nil //nolint:nilnil // Intentional: fx dependency injection allows nil values
	}

	return NewEventSender(settings, lifecycle)
}

// NewEventSender creates a new NATS event sender and manages its lifecycle.
func NewEventSender(
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

// NewEventReceiverOptional creates a new NATS event receiver only when events are enabled.
func NewEventReceiverOptional(
	settings *config.EventSettings,
	lifecycle fx.Lifecycle,
) (*cenats.Consumer, error) {
	if !settings.Enabled {
		// Return nil when events are disabled
		// This is acceptable in fx.Provide context
		return nil, nil //nolint:nilnil // Intentional: fx dependency injection allows nil values
	}

	return NewEventReceiver(settings, lifecycle)
}

// NewEventReceiver creates a new NATS event receiver and manages its lifecycle.
func NewEventReceiver(
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
