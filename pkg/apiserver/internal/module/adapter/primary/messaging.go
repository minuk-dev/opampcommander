package primary

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/fx"

	indirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/messaging/direct"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/messaging/inmemory"
	inkafka "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/messaging/kafka"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter/common"
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
	serverID agentmodel.ServerID,
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
	case config.EventProtocolTypeDirect:
		return createDirectReceiver(settings, serverID, logger)
	case config.EventProtocolTypeInMemory:
		return hub, nil
	}

	return nil, &common.UnsupportedEventProtocolError{
		ProtocolType: settings.ProtocolType.String(),
	}
}

// ErrDirectAddressRequired is returned when the direct transport is configured without the
// listen or advertise address it needs to be reachable.
var ErrDirectAddressRequired = errors.New(
	"direct transport requires event.direct.listenAddress and event.direct.advertiseAddress")

// createDirectReceiver builds the direct (peer-to-peer) receiver, selecting the wire
// sub-protocol (HTTP or gRPC) and binding it to the configured listen address. It fails
// fast when the transport is misconfigured, so a server never starts silently unreachable.
func createDirectReceiver(
	settings *config.EventSettings,
	serverID agentmodel.ServerID,
	logger *slog.Logger,
) (agentport.ServerEventReceiverPort, error) {
	directSettings := settings.DirectSettings
	if directSettings.ListenAddress == "" || directSettings.AdvertiseAddress == "" {
		return nil, ErrDirectAddressRequired
	}

	var receiver indirect.Receiver

	switch directSettings.SubProtocol {
	case config.DirectSubProtocolGRPC:
		receiver = indirect.NewGRPCReceiver(directSettings.ListenAddress, serverID.String(), directSettings.AuthToken, logger)
	case config.DirectSubProtocolHTTP:
		receiver = indirect.NewHTTPReceiver(directSettings.ListenAddress, serverID.String(), directSettings.AuthToken, logger)
	default:
		return nil, &common.UnsupportedEventProtocolError{
			ProtocolType: "direct/" + directSettings.SubProtocol.String(),
		}
	}

	return indirect.NewEventReceiverAdapter(receiver, logger), nil
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
	serverID agentmodel.ServerID,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
) (*inkafka.Consumer, error) {
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

	group, err := sarama.NewConsumerGroup(brokers, groupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka receiver: %w", err)
	}

	consumer := inkafka.NewConsumer(group, topic, logger, settings.KafkaSettings.WithDefaults().RetryBackoff)

	lifecycle.Append(fx.Hook{
		// CloudEvents StartReceiver owns the single OpenInbound loop.
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			err := consumer.Close(ctx)
			if err != nil {
				return fmt.Errorf("failed to close Kafka receiver: %w", err)
			}

			return nil
		},
	})

	return consumer, nil
}
