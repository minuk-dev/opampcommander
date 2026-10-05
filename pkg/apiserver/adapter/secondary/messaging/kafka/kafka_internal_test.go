package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/IBM/sarama"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var errBrokerDown = errors.New("broker down")

var _ protocol.Sender = (*testProtocolSender)(nil)

type testProtocolSender struct {
	attempts            int
	delivered           int
	err                 error
	waitForCancellation bool
}

func (s *testProtocolSender) Send(ctx context.Context, _ binding.Message, _ ...binding.Transformer) error {
	s.attempts++
	if s.waitForCancellation {
		<-ctx.Done()

		return fmt.Errorf("test sender cancelled: %w", ctx.Err())
	}

	if s.err != nil {
		return s.err
	}

	s.delivered++

	return nil
}

func TestNewEventSenderAdapter_DisabledMetrics(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter(t, &testProtocolSender{})
	require.False(t, adapter.Degraded())
}

func TestKafkaConfiguredDeliveryPolicy(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		settings := config.KafkaSettings{
			SendTimeout: 1500 * time.Millisecond, RetryBackoff: 20 * time.Millisecond,
			RetryAttempts: 3, FailureThreshold: 1, ProbeInterval: 4 * time.Second,
		}
		transport := &testProtocolSender{err: errBrokerDown}
		adapter, err := NewEventSenderAdapter(transport, slog.New(slog.DiscardHandler), nil, settings)
		require.NoError(t, err)

		server := &agentmodel.Server{ID: "remote"}
		message := serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache}
		start := time.Now()
		err = adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.Equal(t, settings.RetryAttempts, transport.attempts)
		require.Equal(t, time.Duration(settings.RetryAttempts-1)*settings.RetryBackoff, time.Since(start))

		err = adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.Equal(t, settings.RetryAttempts, transport.attempts, "failure threshold 1 should open the breaker")
		time.Sleep(settings.ProbeInterval - time.Nanosecond)

		err = adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.Equal(t, settings.RetryAttempts, transport.attempts, "probe must wait for the configured cooldown")

		transport.err = nil

		time.Sleep(time.Nanosecond)
		require.NoError(t, adapter.SendMessageToServer(t.Context(), server, message))
		require.Equal(t, settings.RetryAttempts+1, transport.attempts)
		require.False(t, adapter.Degraded())

		transport.waitForCancellation = true
		start = time.Now()
		err = adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.Equal(t, settings.SendTimeout, time.Since(start))
	})
}

func TestKafkaFailureIsNotReplayed(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		transport := &testProtocolSender{err: errBrokerDown}
		adapter := newTestAdapter(t, transport)
		message := serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache}
		server := &agentmodel.Server{ID: "remote"}
		err := adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.True(t, adapter.Degraded())

		before := transport.attempts
		transport.err = nil

		time.Sleep(10 * time.Second)
		require.Equal(t, before, transport.attempts, "recovery must not retry an old event")
		require.Zero(t, transport.delivered)
		require.NoError(t, adapter.SendMessageToServer(t.Context(), server, message))
		require.Equal(t, 1, transport.delivered)
		require.False(t, adapter.Degraded())
	})
}

func TestKafkaBreakerFailsFast(t *testing.T) {
	t.Parallel()

	transport := &testProtocolSender{err: errBrokerDown}
	adapter := newTestAdapter(t, transport)
	message := serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache}
	server := &agentmodel.Server{ID: "remote"}

	for range 2 {
		err := adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
	}

	before := transport.attempts
	err := adapter.SendMessageToServer(t.Context(), server, message)
	require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
	require.Equal(t, before, transport.attempts, "open breaker should skip the broker")
}

func TestKafkaPermanentErrorDoesNotOpenBreaker(t *testing.T) {
	t.Parallel()

	transport := &testProtocolSender{err: sarama.ConfigurationError("message larger than Producer.MaxMessageBytes")}
	adapter := newTestAdapter(t, transport)
	err := adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "oversized"},
		serverevent.Message{Target: "oversized", Type: serverevent.MessageTypeInvalidateAgentCache})

	var configurationError sarama.ConfigurationError
	require.ErrorAs(t, err, &configurationError)
	require.NotErrorIs(t, err, model.ErrTargetServerUnreachable)
	require.False(t, adapter.Degraded())
	require.Equal(t, 1, transport.attempts, "permanent errors must not be retried")
	transport.err = nil

	require.NoError(t, adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "valid"},
		serverevent.Message{Target: "valid", Type: serverevent.MessageTypeInvalidateAgentCache}))
	require.Equal(t, 2, transport.attempts)
}

func newTestAdapter(t *testing.T, sender protocol.Sender) *EventSenderAdapter {
	t.Helper()

	adapter, err := NewEventSenderAdapter(sender, slog.New(slog.DiscardHandler), nil, config.KafkaSettings{})
	require.NoError(t, err)

	return adapter
}
