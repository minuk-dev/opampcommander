package kafka

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
)

type testConsumerGroup struct {
	sarama.ConsumerGroup

	consume func(context.Context) error
	errors  chan error
	closed  bool
}

func (g *testConsumerGroup) Consume(ctx context.Context, _ []string, _ sarama.ConsumerGroupHandler) error {
	return g.consume(ctx)
}
func (g *testConsumerGroup) Errors() <-chan error { return g.errors }
func (g *testConsumerGroup) Close() error {
	g.closed = true
	close(g.errors)

	return nil
}

func TestConsumer_RecoversWithoutClosingReceiver(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		var attempts []time.Time

		group := &testConsumerGroup{errors: make(chan error), consume: func(ctx context.Context) error {
			attempts = append(attempts, time.Now())
			switch len(attempts) {
			case 1:
				return sarama.ErrOutOfBrokers
			case 2:
				return nil // normal rebalance, no delay.
			default:
				<-ctx.Done()

				return ctx.Err()
			}
		}}
		consumer := NewConsumer(group, "events", slog.New(slog.DiscardHandler), time.Second)

		done := make(chan error, 1)
		go func() { done <- consumer.OpenInbound(ctx) }()

		received := make(chan error, 1)

		go func() { _, err := consumer.Receive(ctx); received <- err }()

		time.Sleep(time.Second)
		synctest.Wait()
		require.Len(t, attempts, 3)
		require.Equal(t, time.Second, attempts[1].Sub(attempts[0]))
		require.Equal(t, attempts[1], attempts[2])
		require.Empty(t, received, "a broker failure must not close the CloudEvents receiver")

		group.errors <- sarama.ErrOutOfBrokers // asynchronous errors must also be drained.

		cancel()
		require.NoError(t, <-done)
		require.ErrorIs(t, <-received, io.EOF)
		require.NoError(t, consumer.Close(t.Context()))
		require.True(t, group.closed)
	})
}

func TestConsumer_CancelBackoff(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		group := &testConsumerGroup{errors: make(chan error), consume: func(context.Context) error {
			return sarama.ErrOutOfBrokers
		}}
		consumer := NewConsumer(group, "events", slog.New(slog.DiscardHandler), time.Hour)

		done := make(chan error, 1)
		go func() { done <- consumer.OpenInbound(ctx) }()

		synctest.Wait()
		cancel()
		require.NoError(t, <-done)
		require.NoError(t, consumer.Close(t.Context()))
	})
}
