package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	"github.com/cloudevents/sdk-go/v2/protocol"
)

var (
	_ protocol.Receiver = (*Consumer)(nil)
	_ protocol.Opener   = (*Consumer)(nil)
	_ protocol.Closer   = (*Consumer)(nil)
)

// Consumer keeps the CloudEvents receiver open across transient consumer-group errors.
type Consumer struct {
	*cekafka.Receiver

	group        sarama.ConsumerGroup
	topic        string
	logger       *slog.Logger
	retryBackoff time.Duration
}

// NewConsumer reuses CloudEvents decoding/acknowledgements and Sarama group management.
func NewConsumer(group sarama.ConsumerGroup, topic string, logger *slog.Logger, retryBackoff time.Duration) *Consumer {
	return &Consumer{Receiver: cekafka.NewReceiver(), group: group, topic: topic,
		logger: logger, retryBackoff: retryBackoff}
}

// OpenInbound retries consumption without closing the receiver on broker loss.
func (c *Consumer) OpenInbound(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go c.drainErrors(ctx)

	for ctx.Err() == nil {
		err := c.group.Consume(ctx, []string{c.topic}, c.Receiver)
		if ctx.Err() != nil || errors.Is(err, sarama.ErrClosedConsumerGroup) || errors.Is(err, sarama.ErrClosedClient) {
			return nil
		}

		if err == nil {
			continue // a normal rebalance starts a new session immediately.
		}

		c.logger.Warn("Kafka consumption failed; retrying", "topic", c.topic, "error", err)

		timer := time.NewTimer(c.retryBackoff)
		select {
		case <-ctx.Done():
			timer.Stop()

			return nil
		case <-timer.C:
		}
	}

	return nil
}

// Close stops consumption before closing the CloudEvents incoming channel.
func (c *Consumer) Close(ctx context.Context) error {
	err := c.group.Close()
	if err != nil {
		return fmt.Errorf("close Kafka consumer group: %w", err)
	}

	err = c.Receiver.Close(ctx)
	if err != nil {
		return fmt.Errorf("close Kafka receiver: %w", err)
	}

	return nil
}

func (c *Consumer) drainErrors(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-c.group.Errors():
			if !ok {
				return
			}

			c.logger.Warn("Kafka consumer error", "topic", c.topic, "error", err)
		}
	}
}
