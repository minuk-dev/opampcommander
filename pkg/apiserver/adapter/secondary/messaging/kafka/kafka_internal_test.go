package kafka

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var errBrokerDown = errors.New("broker down")

func TestNewEventSenderAdapter_DisabledMetrics(t *testing.T) {
	t.Parallel()

	adapter, err := NewEventSenderAdapter(&cekafka.Sender{}, slog.New(slog.DiscardHandler), nil)
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

func newTestAdapter(t *testing.T) *EventSenderAdapter {
	t.Helper()

	adapter, err := NewEventSenderAdapter(&cekafka.Sender{}, slog.New(slog.DiscardHandler), nil)
	require.NoError(t, err)

	return adapter
}
