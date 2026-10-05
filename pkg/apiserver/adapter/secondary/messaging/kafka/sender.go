package kafka

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/protocol"
)

var _ protocol.Sender = (*Sender)(nil)

var errEventEncoding = errors.New("invalid Kafka event encoding")

// Sender waits for asynchronous Kafka acknowledgements with a cancellable context.
// One shared goroutine drains acknowledgements, including those arriving after a timeout.
type Sender struct {
	producer  sarama.AsyncProducer
	topic     string
	done      chan struct{}
	closeOnce sync.Once
}

// NewSender requires both Producer.Return.Successes and Producer.Return.Errors enabled.
func NewSender(producer sarama.AsyncProducer, topic string) *Sender {
	sender := &Sender{producer: producer, topic: topic, done: make(chan struct{}), closeOnce: sync.Once{}}
	go sender.drain()

	return sender
}

// Send encodes a CloudEvent and waits until Kafka accepts it or the context expires.
// Kafka can still accept an already submitted event after cancellation; retries may duplicate it.
func (s *Sender) Send(ctx context.Context, message binding.Message, transformers ...binding.Transformer) error {
	if ctx.Err() != nil {
		return fmt.Errorf("send Kafka event: %w", ctx.Err())
	}

	result := make(chan error, 1)
	kafkaMessage := &sarama.ProducerMessage{}
	kafkaMessage.Topic = s.topic
	kafkaMessage.Metadata = result

	err := cekafka.WriteProducerMessage(ctx, message, kafkaMessage, transformers...)
	if err != nil {
		return fmt.Errorf("%w: %w", errEventEncoding, err)
	}

	select {
	case s.producer.Input() <- kafkaMessage:
	case <-ctx.Done():
		return fmt.Errorf("submit Kafka event: %w", ctx.Err())
	case <-s.done:
		return sarama.ErrClosedClient
	}

	select {
	case err = <-result:
		if err != nil {
			return fmt.Errorf("kafka acknowledgement: %w", err)
		}

		return nil
	case <-ctx.Done():
		return fmt.Errorf("await Kafka acknowledgement: %w", ctx.Err())
	case <-s.done:
		return sarama.ErrClosedClient
	}
}

// Close initiates shutdown while continuing to drain results; waiting is bounded by ctx.
func (s *Sender) Close(ctx context.Context) error {
	s.closeOnce.Do(s.producer.AsyncClose)

	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("close Kafka producer: %w", ctx.Err())
	}
}

func (s *Sender) drain() {
	defer close(s.done)

	successes, failures := s.producer.Successes(), s.producer.Errors()
	for successes != nil || failures != nil {
		select {
		case message, ok := <-successes:
			if !ok {
				successes = nil

				continue
			}

			acknowledge(message, nil)
		case failure, ok := <-failures:
			if !ok {
				failures = nil

				continue
			}

			acknowledge(failure.Msg, failure.Err)
		}
	}
}

func acknowledge(message *sarama.ProducerMessage, err error) {
	result, ok := message.Metadata.(chan error)
	if ok {
		result <- err
	}
}
