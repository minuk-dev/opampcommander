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

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var errBrokerDown = errors.New("broker down")

func TestNewEventSenderAdapter_DisabledMetrics(t *testing.T) {
	t.Parallel()

	adapter, err := NewEventSenderAdapter(&Sender{}, slog.New(slog.DiscardHandler), nil)
	require.NoError(t, err)
	require.False(t, adapter.Degraded())
}

func TestKafkaFailureQueuesAndReplays(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter(t)

	var (
		available atomic.Bool
		delivered atomic.Int32
	)

	adapter.send = func(context.Context, cloudevents.Event) error {
		if !available.Load() {
			return errBrokerDown
		}

		delivered.Add(1)

		return nil
	}
	message := serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache}

	err := adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "remote"}, message)
	require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
	require.True(t, adapter.Degraded())

	available.Store(true)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		defer close(done)

		adapter.Replay(ctx)
	}()

	t.Cleanup(func() { cancel(); <-done })

	require.Eventually(t, func() bool {
		return delivered.Load() == 1 && !adapter.Degraded()
	}, 3*time.Second, 10*time.Millisecond, "queued event should replay after recovery")
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

func TestKafkaPermanentErrorDoesNotBlockReplay(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		adapter := newTestAdapter(t)
		available := false
		delivered := 0

		adapter.send = func(_ context.Context, event cloudevents.Event) error {
			if !available {
				return errBrokerDown
			}

			if event.Subject() == "oversized" {
				return sarama.ConfigurationError("message larger than Producer.MaxMessageBytes")
			}

			delivered++

			return nil
		}
		for _, target := range []string{"oversized", "valid"} {
			err := adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: target},
				serverevent.Message{Target: target, Type: serverevent.MessageTypeInvalidateAgentCache})
			require.ErrorIs(t, err, model.ErrTargetServerUnreachable)
		}

		available = true
		ctx, cancel := context.WithCancel(t.Context())

		done := make(chan struct{})
		go func() { defer close(done); adapter.Replay(ctx) }()

		time.Sleep(8 * time.Second)
		cancel()
		<-done
		require.Equal(t, 1, delivered)
		require.False(t, adapter.Degraded())
		// A new permanent failure is rejected immediately without opening the breaker or queueing.
		err := adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "oversized"},
			serverevent.Message{Target: "oversized", Type: serverevent.MessageTypeInvalidateAgentCache})

		var configurationError sarama.ConfigurationError
		require.ErrorAs(t, err, &configurationError)
		require.NotErrorIs(t, err, model.ErrTargetServerUnreachable)
		require.False(t, adapter.Degraded())
	})
}

func newTestAdapter(t *testing.T) *EventSenderAdapter {
	t.Helper()

	adapter, err := NewEventSenderAdapter(&Sender{}, slog.New(slog.DiscardHandler), nil)
	require.NoError(t, err)

	return adapter
}
