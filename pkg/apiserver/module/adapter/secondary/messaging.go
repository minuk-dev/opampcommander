package secondary

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/inmemory"
	outkafka "github.com/minuk-dev/opampcommander/internal/adapter/out/messaging/kafka"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/adapter/common"
)

const (
	// defaultSenderCloseTimeout is the timeout for closing the Kafka sender.
	defaultSenderCloseTimeout = 10 * time.Second

	// defaultKafkaTimeout is the timeout for Kafka metadata operations.
	defaultKafkaTimeout = 30 * time.Second

	// defaultKafkaRetryMax is the maximum number of retries for Kafka metadata operations.
	defaultKafkaRetryMax = 10

	// defaultKafkaRetryBackoff is the backoff duration between Kafka metadata retries.
	defaultKafkaRetryBackoff = 2 * time.Second
)

// newEventSender provides the outbound server-event sender, selecting the transport
// based on the configured event protocol. In standalone (in-memory) mode it returns
// the shared hub so events are observed by the receiver in the primary adapter.
func newEventSender(
	settings *config.EventSettings,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
	hub *inmemory.EventSenderAdapter,
) (agentport.ServerEventSenderPort, error) {
	switch settings.ProtocolType {
	case config.EventProtocolTypeKafka:
		sender, err := createKafkaSender(settings, lifecycle)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kafka sender: %w", err)
		}

		adapter, err := outkafka.NewEventSenderAdapter(sender, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kafka event sender adapter: %w", err)
		}

		return adapter, nil
	case config.EventProtocolTypeInMemory:
		return hub, nil
	}

	return nil, &common.UnsupportedEventProtocolError{
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
	saramaConfig.Metadata.Timeout = defaultKafkaTimeout
	saramaConfig.Metadata.Retry.Max = defaultKafkaRetryMax
	saramaConfig.Metadata.Retry.Backoff = defaultKafkaRetryBackoff
	topic := settings.KafkaSettings.Topic

	var opts []cekafka.SenderOptionFunc

	sender, err := cekafka.NewSender(brokers, saramaConfig, topic, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka sender: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, defaultSenderCloseTimeout)
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
