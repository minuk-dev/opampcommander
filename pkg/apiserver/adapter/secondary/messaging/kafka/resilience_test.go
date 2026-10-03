package kafka

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.opentelemetry.io/otel/metric/noop"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

func TestKafkaFailureQueuesAndReplays(t *testing.T) {
	t.Parallel()
	meter := noop.NewMeterProvider().Meter("test")
	sent, _ := meter.Int64Counter("sent")
	failed, _ := meter.Int64Counter("failed")
	dropped, _ := meter.Int64Counter("dropped")
	state, _ := meter.Int64Gauge("state")
	var available atomic.Bool
	var delivered atomic.Int32
	adapter := &EventSenderAdapter{
		send: func(context.Context, cloudevents.Event) error {
			if !available.Load() {
				return errors.New("broker down")
			}
			delivered.Add(1)
			return nil
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		clock:  clock.NewRealClock(),
		sent:   sent, failed: failed, dropped: dropped, state: state,
	}
	message := serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache}
	err := adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "remote"}, message)
	if !errors.Is(err, model.ErrTargetServerUnreachable) || !adapter.Degraded() {
		t.Fatalf("expected queued unreachable event, got %v", err)
	}
	available.Store(true)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go adapter.Replay(ctx)
	for delivered.Load() == 0 && ctx.Err() == nil {
		time.Sleep(10 * time.Millisecond)
	}
	if delivered.Load() != 1 || adapter.Degraded() {
		t.Fatalf("expected one replayed event and healthy sender, delivered=%d degraded=%v", delivered.Load(), adapter.Degraded())
	}
}

func TestKafkaBreakerFailsFast(t *testing.T) {
	t.Parallel()
	meter := noop.NewMeterProvider().Meter("test")
	sent, _ := meter.Int64Counter("sent")
	failed, _ := meter.Int64Counter("failed")
	dropped, _ := meter.Int64Counter("dropped")
	state, _ := meter.Int64Gauge("state")
	var attempts atomic.Int32
	adapter := &EventSenderAdapter{
		send:   func(context.Context, cloudevents.Event) error { attempts.Add(1); return errors.New("broker down") },
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)), clock: clock.NewRealClock(),
		sent: sent, failed: failed, dropped: dropped, state: state,
	}
	event := cloudevents.NewEvent()
	for range 2 {
		_ = adapter.deliver(t.Context(), event)
	}
	before := attempts.Load()
	if err := adapter.deliver(t.Context(), event); !errors.Is(err, model.ErrTargetServerUnreachable) || attempts.Load() != before {
		t.Fatalf("breaker did not fail fast: err=%v attempts=%d", err, attempts.Load())
	}
}
