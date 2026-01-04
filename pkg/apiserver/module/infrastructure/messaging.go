package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/inmemory"
	inkafka "github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/kafka"
	outkafka "github.com/minuk-dev/opampcommander/internal/adapter/out/messaging/kafka"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

const (
	// DefaultSenderCloseTimeout is the default timeout for closing the Kafka sender.
	DefaultSenderCloseTimeout = 10 * time.Second

	// DefaultReceiverCloseTimeout is the default timeout for closing the Kafka receiver.
	DefaultReceiverCloseTimeout = 10 * time.Second

	// DefaultKafkaTimeout is the default timeout for Kafka operations.
	DefaultKafkaTimeout = 30 * time.Second

	// DefaultKafkaRetryMax is the default maximum number of retries for Kafka operations.
	DefaultKafkaRetryMax = 10

	// DefaultKafkaRetryBackoff is the default backoff duration between Kafka retries.
	DefaultKafkaRetryBackoff = 2 * time.Second
)

// UnsupportedEventProtocolError is returned when an unsupported event protocol type is specified.
type UnsupportedEventProtocolError struct {
	ProtocolType string
}

// Error implements the error interface.
func (e *UnsupportedEventProtocolError) Error() string {
	return "unsupported event protocol type: " + e.ProtocolType
}

//nolint:ireturn
func newEventSenderAndReceiver(
	settings *config.EventSettings,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
	serverIdentityProvider port.ServerIdentityProvider,
) (port.ServerEventSenderPort, port.ServerEventReceiverPort, error) {
	switch settings.ProtocolType {
	case config.EventProtocolTypeKafka:
		sender, err := createKafkaSender(settings, lifecycle)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Kafka sender: %w", err)
		}

		senderAdapter, err := outkafka.NewEventSenderAdapter(sender, logger)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Kafka event sender adapter: %w", err)
		}

		receiver, err := createKafkaReceiver(settings, logger, lifecycle)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Kafka receiver: %w", err)
		}

		receiverAdapter, err := inkafka.NewEventReceiverAdapter(serverIdentityProvider, receiver, logger)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Kafka event receiver adapter: %w", err)
		}

		return senderAdapter, receiverAdapter, nil
	case config.EventProtocolTypeInMemory:
		adapter := inmemory.NewEventHubAdapter(logger)

		return adapter, adapter, nil
	}

	return nil, nil, &UnsupportedEventProtocolError{
		ProtocolType: settings.ProtocolType.String(),
	}
}

func createKafkaSender(
	settings *config.EventSettings,
	lifecycle fx.Lifecycle,
) (*cekafka.Sender, error) {
	brokers := settings.KafkaSettings.Brokers
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Metadata.Timeout = DefaultKafkaTimeout
	saramaConfig.Metadata.Retry.Max = DefaultKafkaRetryMax
	saramaConfig.Metadata.Retry.Backoff = DefaultKafkaRetryBackoff
	topic := settings.KafkaSettings.Topic

	var opts []cekafka.SenderOptionFunc

	sender, err := cekafka.NewSender(brokers, saramaConfig, topic, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka sender: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, DefaultSenderCloseTimeout)
			defer cancel()

			err := sender.Close(ctx)
			if err != nil {
				return fmt.Errorf("failed to close Kafka sender: %w", err)
			}

			return nil
		},
	})

	return sender, nil
}

// createKafkaReceiver creates a Kafka receiver with lifecycle management.
func createKafkaReceiver(
	settings *config.EventSettings,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
) (*cekafka.Consumer, error) {
	brokers := settings.KafkaSettings.Brokers
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Version = sarama.V2_6_0_0
	saramaConfig.Metadata.Timeout = DefaultKafkaTimeout
	saramaConfig.Metadata.Retry.Max = DefaultKafkaRetryMax
	saramaConfig.Metadata.Retry.Backoff = DefaultKafkaRetryBackoff

	topic := settings.KafkaSettings.Topic
	groupID := "opampcommander-consumer-group"

	consumer, err := cekafka.NewConsumer(brokers, saramaConfig, groupID, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka receiver: %w", err)
	}

	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			// Start OpenInbound asynchronously to avoid blocking application startup
			// OpenInbound may take time to establish connection to Kafka
			go func() {
				err := consumer.OpenInbound(lifecycleCtx)
				if err != nil {
					logger.Error("Kafka receiver OpenInbound error", "error", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			err := consumer.Close(ctx)
			if err != nil {
				return fmt.Errorf("failed to close Kafka receiver: %w", err)
			}

			defer lifecycleCancel()

			return nil
		},
	})

	return consumer, nil
}
