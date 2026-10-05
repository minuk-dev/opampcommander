package kafka

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/IBM/sarama"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var errBrokerDown = errors.New("broker down")

func TestNewEventSenderAdapter_DisabledMetrics(t *testing.T) {
	t.Parallel()

	adapter, err := NewEventSenderAdapter(&Sender{}, slog.New(slog.DiscardHandler), nil, config.KafkaSettings{})
	require.NoError(t, err)
	require.False(t, adapter.Degraded())
}

func TestKafkaConfiguredDeliveryPolicy(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		settings := config.KafkaSettings{
			SendTimeout: 1500 * time.Millisecond, RetryBackoff: 20 * time.Millisecond,
			RetryAttempts: 3, FailureThreshold: 1, ProbeInterval: 4 * time.Second,
		}
		adapter, err := NewEventSenderAdapter(&Sender{}, slog.New(slog.DiscardHandler), nil, settings)
		require.NoError(t, err)

		available := false
		attempts := 0
		adapter.send = func(context.Context, cloudevents.Event) error {
			attempts++

			if !available {
				return errBrokerDown
			}

			return nil
		}
		server := &agentmodel.Server{ID: "remote"}
		message := serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache}
		start := time.Now()
		err = adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.Equal(t, settings.RetryAttempts, attempts)
		require.Equal(t, time.Duration(settings.RetryAttempts-1)*settings.RetryBackoff, time.Since(start))

		err = adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.Equal(t, settings.RetryAttempts, attempts, "failure threshold 1 should open the breaker")
		time.Sleep(settings.ProbeInterval - time.Nanosecond)

		err = adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.Equal(t, settings.RetryAttempts, attempts, "probe must wait for the configured cooldown")

		available = true

		time.Sleep(time.Nanosecond)
		require.NoError(t, adapter.SendMessageToServer(t.Context(), server, message))
		require.Equal(t, settings.RetryAttempts+1, attempts)
		require.False(t, adapter.Degraded())

		adapter.send = func(ctx context.Context, _ cloudevents.Event) error {
			<-ctx.Done()

			return ctx.Err()
		}
		start = time.Now()
		err = adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.Equal(t, settings.SendTimeout, time.Since(start))
	})
}

func TestKafkaFailureIsNotReplayed(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		adapter := newTestAdapter(t)
		available := false
		attempts := 0
		delivered := 0
		adapter.send = func(context.Context, cloudevents.Event) error {
			attempts++

			if !available {
				return errBrokerDown
			}

			delivered++

			return nil
		}
		message := serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache}
		server := &agentmodel.Server{ID: "remote"}
		err := adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.True(t, adapter.Degraded())

		before := attempts
		available = true

		time.Sleep(10 * time.Second)
		require.Equal(t, before, attempts, "recovery must not retry an old event")
		require.Zero(t, delivered)
		require.NoError(t, adapter.SendMessageToServer(t.Context(), server, message))
		require.Equal(t, 1, delivered)
		require.False(t, adapter.Degraded())
	})
}

func TestKafkaBreakerFailsFast(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter(t)

	var attempts atomic.Int32

	adapter.send = func(context.Context, cloudevents.Event) error {
		attempts.Add(1)

		return errBrokerDown
	}
	message := serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache}
	server := &agentmodel.Server{ID: "remote"}

	for range 2 {
		err := adapter.SendMessageToServer(t.Context(), server, message)
		require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
	}

	before := attempts.Load()
	err := adapter.SendMessageToServer(t.Context(), server, message)
	require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
	require.Equal(t, before, attempts.Load(), "open breaker should skip the broker")
}

func TestKafkaPermanentErrorDoesNotOpenBreaker(t *testing.T) {
	t.Parallel()
	adapter := newTestAdapter(t)
	attempts := 0
	adapter.send = func(_ context.Context, event cloudevents.Event) error {
		attempts++

		if event.Subject() == "oversized" {
			return sarama.ConfigurationError("message larger than Producer.MaxMessageBytes")
		}

		return nil
	}
	err := adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "oversized"},
		serverevent.Message{Target: "oversized", Type: serverevent.MessageTypeInvalidateAgentCache})

	var configurationError sarama.ConfigurationError
	require.ErrorAs(t, err, &configurationError)
	require.NotErrorIs(t, err, model.ErrTargetServerUnreachable)
	require.False(t, adapter.Degraded())
	require.Equal(t, 1, attempts, "permanent errors must not be retried")
	require.NoError(t, adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "valid"},
		serverevent.Message{Target: "valid", Type: serverevent.MessageTypeInvalidateAgentCache}))
	require.Equal(t, 2, attempts)
}

func newTestAdapter(t *testing.T) *EventSenderAdapter {
	t.Helper()

	adapter, err := NewEventSenderAdapter(&Sender{}, slog.New(slog.DiscardHandler), nil, config.KafkaSettings{})
	require.NoError(t, err)

	return adapter
}
