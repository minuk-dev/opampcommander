package primary

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/inmemory"
	inkafka "github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/kafka"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/adapter/common"
)

// ErrServerIDRequired is returned when Kafka messaging is configured without a server ID.
var ErrServerIDRequired = errors.New("server ID is required for kafka messaging consumer group")

const (
	// defaultKafkaTimeout is the timeout for Kafka metadata operations.
	defaultKafkaTimeout = 30 * time.Second

	// defaultKafkaRetryMax is the maximum number of retries for Kafka metadata operations.
	defaultKafkaRetryMax = 10

	// defaultKafkaRetryBackoff is the backoff duration between Kafka metadata retries.
	defaultKafkaRetryBackoff = 2 * time.Second
)

// newEventReceiver provides the inbound server-event receiver, selecting the transport
// based on the configured event protocol. In standalone (in-memory) mode it returns
// the shared hub so events sent by the sender in the secondary adapter are observed.
func newEventReceiver(
	settings *config.EventSettings,
	serverID config.ServerID,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
	serverIdentityProvider agentport.ServerIdentityProvider,
	hub *inmemory.EventSenderAdapter,
) (agentport.ServerEventReceiverPort, error) {
	switch settings.ProtocolType {
	case config.EventProtocolTypeKafka:
		receiver, err := createKafkaReceiver(settings, serverID, logger, lifecycle)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kafka receiver: %w", err)
		}

		adapter, err := inkafka.NewEventReceiverAdapter(serverIdentityProvider, receiver, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kafka event receiver adapter: %w", err)
		}

		return adapter, nil
	case config.EventProtocolTypeInMemory:
		return hub, nil
	}

	return nil, &common.UnsupportedEventProtocolError{
		ProtocolType: settings.ProtocolType.String(),
	}
}

// createKafkaReceiver creates a Kafka receiver with lifecycle management.
//
// Each server uses its own consumer group so that every server receives every
// inter-server event (broadcast). The receiver then filters by event.Subject
// to keep only those targeting this server. A single shared consumer group
// would route each event to exactly one consumer, silently dropping events
// addressed to other servers.
func createKafkaReceiver(
	settings *config.EventSettings,
	serverID config.ServerID,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
) (*cekafka.Consumer, error) {
	if serverID.String() == "" {
		return nil, ErrServerIDRequired
	}

	brokers := settings.KafkaSettings.Brokers
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Version = sarama.V2_6_0_0
	saramaConfig.Metadata.Timeout = defaultKafkaTimeout
	saramaConfig.Metadata.Retry.Max = defaultKafkaRetryMax
	saramaConfig.Metadata.Retry.Backoff = defaultKafkaRetryBackoff

	topic := settings.KafkaSettings.Topic
	groupID := "opampcommander-consumer-group-" + serverID.String()

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
			// Cancel the lifecycle context first so OpenInbound stops receiving,
			// then close the underlying consumer to release resources.
			lifecycleCancel()

			err := consumer.Close(ctx)
			if err != nil {
				return fmt.Errorf("failed to close Kafka receiver: %w", err)
			}

			return nil
		},
	})

	return consumer, nil
}
