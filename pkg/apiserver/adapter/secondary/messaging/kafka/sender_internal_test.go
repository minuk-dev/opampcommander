package kafka

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

type testAsyncProducer struct {
	sarama.AsyncProducer

	input     chan *sarama.ProducerMessage
	failures  chan *sarama.ProducerError
	holdClose bool
}

func (p *testAsyncProducer) Input() chan<- *sarama.ProducerMessage { return p.input }
func (p *testAsyncProducer) Errors() <-chan *sarama.ProducerError  { return p.failures }
func (p *testAsyncProducer) AsyncClose() {
	if !p.holdClose {
		close(p.failures)
	}
}

func TestSender_EnqueueIsBoundedAndDoesNotWaitForAck(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"blocked input", "accepted input"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				capacity := 0
				if name == "accepted input" {
					capacity = 1
				}

				producer := &testAsyncProducer{input: make(chan *sarama.ProducerMessage, capacity),
					failures: make(chan *sarama.ProducerError)}
				timeout := 50 * time.Millisecond

				sender := NewSender(producer, "events", slog.New(slog.DiscardHandler), timeout)
				defer func() { require.NoError(t, sender.Close(t.Context())) }()

				adapter, err := NewEventSenderAdapter(sender)
				require.NoError(t, err)

				start := time.Now()

				err = adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "remote"},
					serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache})
				if capacity == 0 {
					require.ErrorIs(t, err, context.DeadlineExceeded)
					require.Equal(t, timeout, time.Since(start))
				} else {
					require.NoError(t, err)
					require.Zero(t, time.Since(start), "enqueue must not wait for a Kafka acknowledgement")
					require.Nil(t, (<-producer.input).Metadata, "no per-message acknowledgement channel")
				}
			})
		})
	}
}

func TestSender_LogsAsyncFailure(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		producer := &testAsyncProducer{input: make(chan *sarama.ProducerMessage, 1),
			failures: make(chan *sarama.ProducerError)}

		var logs bytes.Buffer

		sender := NewSender(producer, "events", slog.New(slog.NewTextHandler(&logs, nil)), time.Second)
		defer func() { require.NoError(t, sender.Close(t.Context())) }()

		adapter, err := NewEventSenderAdapter(sender)
		require.NoError(t, err)
		require.NoError(t, adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "remote"},
			serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache}))

		producer.failures <- &sarama.ProducerError{Msg: <-producer.input, Err: sarama.ErrMessageSizeTooLarge}

		synctest.Wait()
		require.Contains(t, logs.String(), "Kafka notification delivery failed")
		require.Contains(t, logs.String(), sarama.ErrMessageSizeTooLarge.Error())
	})
}

func TestSender_ShutdownIsBoundedAndRejectsClosedProducer(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		producer := &testAsyncProducer{input: make(chan *sarama.ProducerMessage, 1),
			failures: make(chan *sarama.ProducerError), holdClose: true}
		sender := NewSender(producer, "events", slog.New(slog.DiscardHandler), time.Second)

		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		require.ErrorIs(t, sender.Close(ctx), context.DeadlineExceeded)
		// The producer can finish draining after the shutdown deadline.
		close(producer.failures)
		synctest.Wait()
		require.NoError(t, sender.Close(t.Context()))
		adapter, err := NewEventSenderAdapter(sender)
		require.NoError(t, err)
		require.ErrorIs(t, adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "remote"},
			serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache}), sarama.ErrClosedClient)
	})
}
