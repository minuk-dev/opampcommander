package secondary

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/messaging/inmemory"
	outdirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/messaging/direct"
	outkafka "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/messaging/kafka"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter/common"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

// EventSenderResult registers producer health only for the Kafka transport.
type EventSenderResult struct {
	fx.Out

	Sender     agentport.ServerEventSenderPort
	Indicators []healthcheck.HealthIndicator `group:"health_indicators,flatten"`
}

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
	meterProvider metric.MeterProvider,
) (EventSenderResult, error) {
	switch settings.ProtocolType {
	case config.EventProtocolTypeKafka:
		sender, err := createKafkaSender(settings, lifecycle, logger, meterProvider)
		if err != nil {
			return EventSenderResult{}, fmt.Errorf("failed to create Kafka sender: %w", err)
		}

		adapter, err := outkafka.NewEventSenderAdapter(sender)
		if err != nil {
			return EventSenderResult{}, fmt.Errorf("failed to create Kafka event sender adapter: %w", err)
		}

		//exhaustruct:ignore // fx.Out is a marker.
		return EventSenderResult{Sender: adapter, Indicators: []healthcheck.HealthIndicator{sender}}, nil
	case config.EventProtocolTypeDirect:
		sender, err := createDirectSender(settings, logger, lifecycle)
		//exhaustruct:ignore // no Kafka health indicator for direct transport.
		return EventSenderResult{Sender: sender}, err
	case config.EventProtocolTypeInMemory:
		//exhaustruct:ignore // no Kafka health indicator for in-memory transport.
		return EventSenderResult{Sender: hub}, nil
	}

	return EventSenderResult{}, &common.UnsupportedEventProtocolError{
		ProtocolType: settings.ProtocolType.String(),
	}
}

// createDirectSender builds the direct (peer-to-peer) sender, selecting the wire
// sub-protocol (HTTP or gRPC) and registering the deliverer for graceful shutdown.
func createDirectSender(
	settings *config.EventSettings,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
) (agentport.ServerEventSenderPort, error) {
	token := settings.DirectSettings.AuthToken

	var deliverer outdirect.Deliverer

	switch settings.DirectSettings.SubProtocol {
	case config.DirectSubProtocolGRPC:
		deliverer = outdirect.NewGRPCDeliverer(token, logger)
	case config.DirectSubProtocolHTTP:
		deliverer = outdirect.NewHTTPDeliverer(token, logger)
	default:
		return nil, &common.UnsupportedEventProtocolError{
			ProtocolType: "direct/" + settings.DirectSettings.SubProtocol.String(),
		}
	}

	adapter := outdirect.NewEventSenderAdapter(deliverer, logger)

	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(context.Context) error {
			err := adapter.Close()
			if err != nil {
				return fmt.Errorf("failed to close direct sender: %w", err)
			}

			return nil
		},
	})

	return adapter, nil
}

func createKafkaSender(
	settings *config.EventSettings,
	lifecycle fx.Lifecycle,
	logger *slog.Logger,
	meterProvider metric.MeterProvider,
) (*outkafka.Sender, error) {
	kafkaSettings := settings.KafkaSettings.WithDefaults()

	err := kafkaSettings.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid Kafka delivery settings: %w", err)
	}

	brokers := kafkaSettings.Brokers
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Timeout = kafkaSettings.SendTimeout
	saramaConfig.Producer.Retry.Max = kafkaSettings.RetryAttempts - 1
	saramaConfig.Producer.Retry.Backoff = kafkaSettings.RetryBackoff
	saramaConfig.Net.DialTimeout = kafkaSettings.SendTimeout
	saramaConfig.Net.ReadTimeout = kafkaSettings.SendTimeout
	saramaConfig.Net.WriteTimeout = kafkaSettings.SendTimeout
	saramaConfig.Metadata.Timeout = defaultKafkaTimeout
	saramaConfig.Metadata.Retry.Max = defaultKafkaRetryMax
	saramaConfig.Metadata.Retry.Backoff = defaultKafkaRetryBackoff
	topic := kafkaSettings.Topic

	producer, err := sarama.NewAsyncProducer(brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka sender: %w", err)
	}

	sender := outkafka.NewSender(producer, topic, logger, kafkaSettings.SendTimeout)
	if meterProvider != nil {
		err = sender.Observe(meterProvider.Meter("github.com/minuk-dev/opampcommander/kafka"))
		if err != nil {
			logger.Warn("failed to build Kafka delivery metrics; continuing without them", "error", err)
		}
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
