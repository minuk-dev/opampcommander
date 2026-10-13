package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/protocol"
)

var _ protocol.Sender = (*Sender)(nil)

// Sender enqueues best-effort Kafka notifications without waiting for acknowledgements.
type Sender struct {
	producer    sarama.AsyncProducer
	topic       string
	logger      *slog.Logger
	sendTimeout time.Duration
	done        chan struct{}
	closeOnce   sync.Once
	enqueued    atomic.Int64
	delivered   atomic.Int64
	failed      atomic.Int64
	rejected    atomic.Int64
	lastFailure atomic.Pointer[string]
}

// NewSender requires Producer.Return.Errors and Producer.Return.Successes enabled.
func NewSender(producer sarama.AsyncProducer, topic string, logger *slog.Logger, sendTimeout time.Duration) *Sender {
	//exhaustruct:ignore // counters and the last failure start empty.
	sender := &Sender{producer: producer, topic: topic, logger: logger, sendTimeout: sendTimeout,
		done: make(chan struct{}), closeOnce: sync.Once{}}
	go sender.drainResults()

	return sender
}

// Send bounds enqueue time. Success means submitted to the producer, not delivered to Kafka.
func (s *Sender) Send(ctx context.Context, message binding.Message, transformers ...binding.Transformer) (err error) {
	defer func() {
		if err != nil {
			s.rejected.Add(1)
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, s.sendTimeout)
	defer cancel()

	if ctx.Err() != nil {
		return fmt.Errorf("submit Kafka event: %w", ctx.Err())
	}

	kafkaMessage := &sarama.ProducerMessage{}

	kafkaMessage.Topic = s.topic

	err = cekafka.WriteProducerMessage(ctx, message, kafkaMessage, transformers...)
	if err != nil {
		return fmt.Errorf("encode Kafka event: %w", err)
	}

	select {
	case <-s.done:
		return sarama.ErrClosedClient
	default:
	}

	select {
	case s.producer.Input() <- kafkaMessage:
		s.enqueued.Add(1)

		return nil
	case <-ctx.Done():
		return fmt.Errorf("submit Kafka event: %w", ctx.Err())
	case <-s.done:
		return sarama.ErrClosedClient
	}
}

// Close asks Sarama to flush pending messages while continuing to drain errors.
// Waiting is bounded by ctx; Sarama may finish after that deadline.
func (s *Sender) Close(ctx context.Context) error {
	s.closeOnce.Do(s.producer.AsyncClose)

	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("close Kafka producer: %w", ctx.Err())
	}
}

func (s *Sender) drainResults() {
	defer close(s.done)

	failures, successes := s.producer.Errors(), s.producer.Successes()
	for failures != nil || successes != nil {
		select {
		case failure, ok := <-failures:
			if !ok {
				failures = nil

				continue
			}

			s.failed.Add(1)

			reason := failure.Err.Error()
			s.lastFailure.Store(&reason)
			s.logger.Warn("Kafka notification delivery failed", "topic", s.topic, "error", failure.Err)
		case _, ok := <-successes:
			if !ok {
				successes = nil

				continue
			}

			s.delivered.Add(1)
			s.lastFailure.Store(nil)
		}
	}
}
