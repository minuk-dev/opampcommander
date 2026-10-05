package kafka

import (
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

type testAsyncProducer struct {
	sarama.AsyncProducer

	input     chan *sarama.ProducerMessage
	successes chan *sarama.ProducerMessage
	failures  chan *sarama.ProducerError
}

func (p *testAsyncProducer) Input() chan<- *sarama.ProducerMessage     { return p.input }
func (p *testAsyncProducer) Successes() <-chan *sarama.ProducerMessage { return p.successes }
func (p *testAsyncProducer) Errors() <-chan *sarama.ProducerError      { return p.failures }
func (p *testAsyncProducer) AsyncClose() {
	close(p.successes)
	close(p.failures)
}

func TestSender_BoundsSubmissionAndAcknowledgement(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"blocked input", "late acknowledgement"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				producer := &testAsyncProducer{
					input:     make(chan *sarama.ProducerMessage),
					successes: make(chan *sarama.ProducerMessage),
					failures:  make(chan *sarama.ProducerError),
				}
				sender := NewSender(producer, "events")
				adapter, err := NewEventSenderAdapter(sender, slog.New(slog.DiscardHandler), nil, config.KafkaSettings{})
				require.NoError(t, err)

				if name == "late acknowledgement" {
					go func() {
						message := <-producer.input

						time.Sleep(3 * time.Second)

						producer.successes <- message
					}()
				}

				start := time.Now()
				err = adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "remote"},
					serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache})
				require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
				require.Equal(t, config.DefaultKafkaSettings().SendTimeout, time.Since(start))
				require.True(t, adapter.Degraded())

				if name == "late acknowledgement" {
					time.Sleep(time.Second)
					synctest.Wait()
					// A late acknowledgement must not block the shared drain goroutine.
					go func() { producer.successes <- <-producer.input }()

					err = adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "remote"},
						serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache})
					require.NoError(t, err)
				}

				require.NoError(t, sender.Close(t.Context()))
			})
		})
	}
}

func TestSender_PropagatesPermanentProducerFailure(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		producer := &testAsyncProducer{
			input:     make(chan *sarama.ProducerMessage),
			successes: make(chan *sarama.ProducerMessage),
			failures:  make(chan *sarama.ProducerError),
		}
		sender := NewSender(producer, "events")
		adapter, err := NewEventSenderAdapter(sender, slog.New(slog.DiscardHandler), nil, config.KafkaSettings{})
		require.NoError(t, err)

		go func() {
			message := <-producer.input
			producer.failures <- &sarama.ProducerError{Msg: message, Err: sarama.ErrMessageSizeTooLarge}
		}()

		err = adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "remote"},
			serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache})
		require.ErrorIs(t, err, sarama.ErrMessageSizeTooLarge)
		require.NotErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.False(t, adapter.Degraded())
		require.NoError(t, sender.Close(t.Context()))
	})
}
