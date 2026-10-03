// Package kafka implements Kafka messaging adapters.
package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	observabilityClient "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/client"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"

	kafkamodel "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/kafka"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var (
	_ agentport.ServerEventSenderPort = (*EventSenderAdapter)(nil)
)

// EventSenderAdapter implements agentport.ServerEventSenderPort using Kafka CloudEvents sender.
type EventSenderAdapter struct {
	send      func(context.Context, cloudevents.Event) error
	logger    *slog.Logger
	clock     clock.Clock
	mu        sync.Mutex
	queue     []cloudevents.Event
	failures  int
	openUntil time.Time
	probing   bool
	sent      metric.Int64Counter
	failed    metric.Int64Counter
	dropped   metric.Int64Counter
	state     metric.Int64Gauge
}

// NewEventSenderAdapter creates a new EventSenderAdapter.
func NewEventSenderAdapter(
	protocolSender *cekafka.Sender,
	logger *slog.Logger,
	meterProvider metric.MeterProvider,
) (*EventSenderAdapter, error) {
	//nolint:godox
	// TODO: cloudevents's observability does not support to inject TracerProvider instead of global
	// https://github.com/cloudevents/sdk-go/pull/1202
	otelService := observabilityClient.NewOTelObservabilityService()

	opts := make([]client.Option, 0, 1)

	opts = append(opts, client.WithObservabilityService(otelService))

	sender, err := cloudevents.NewClient(protocolSender, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudEvents client for sender: %w", err)
	}

	// sender can be nil when events are disabled
	meter := meterProvider.Meter("opampcommander/kafka-events")
	sent, err := meter.Int64Counter("opampcommander.kafka.events.sent")
	if err != nil {
		return nil, fmt.Errorf("create sent counter: %w", err)
	}
	failed, err := meter.Int64Counter("opampcommander.kafka.events.failed")
	if err != nil {
		return nil, fmt.Errorf("create failed counter: %w", err)
	}
	dropped, err := meter.Int64Counter("opampcommander.kafka.events.dropped")
	if err != nil {
		return nil, fmt.Errorf("create dropped counter: %w", err)
	}
	state, err := meter.Int64Gauge("opampcommander.kafka.breaker.open")
	if err != nil {
		return nil, fmt.Errorf("create breaker gauge: %w", err)
	}
	return &EventSenderAdapter{
		send:   func(ctx context.Context, event cloudevents.Event) error { return sender.Send(ctx, event) },
		logger: logger,
		clock:  clock.NewRealClock(),
		sent:   sent, failed: failed, dropped: dropped, state: state,
	}, nil
}

// SendMessageToServer implements agentport.ServerEventSenderPort.
func (e *EventSenderAdapter) SendMessageToServer(
	ctx context.Context,
	server *agentmodel.Server,
	message serverevent.Message,
) error {
	serverID := server.ID
	event := cloudevents.NewEvent()

	// Is it better to add message.EventID field?
	eventID := uuid.New().String()
	event.SetID(eventID)
	event.SetSource(newSource(serverID))
	event.SetSubject(message.Target) // use targetServer as subject
	event.SetType(kafkamodel.EventTypeFromMessageType(message.Type))
	event.SetSpecVersion(kafkamodel.CloudEventMessageSpec)
	event.SetTime(e.clock.Now())

	err := event.SetData(kafkamodel.CloudEventContentType, message.Payload)
	if err != nil {
		return fmt.Errorf("failed to set event data for server %s: %w", serverID, err)
	}

	err = e.deliver(ctx, event)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !e.enqueue(ctx, event) {
			return fmt.Errorf("server %s: %w: replay queue full (event dropped)", serverID, model.ErrTargetServerUnreachable)
		}
		return fmt.Errorf("server %s: %w: %v", serverID, model.ErrTargetServerUnreachable, err)
	}

	return nil
}

const queueLimit = 256

// deliver bounds each request and opens the breaker after two failed attempts.
func (e *EventSenderAdapter) deliver(ctx context.Context, event cloudevents.Event) error {
	e.mu.Lock()
	if e.probing || time.Now().Before(e.openUntil) {
		e.mu.Unlock()
		e.failed.Add(ctx, 1)
		return model.ErrTargetServerUnreachable
	}
	if !e.openUntil.IsZero() {
		e.probing = true
	}
	e.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var err error
	for attempt := range 2 {
		if err = e.send(ctx, event); err == nil {
			e.mu.Lock()
			e.failures = 0
			e.openUntil = time.Time{}
			e.probing = false
			e.mu.Unlock()
			e.state.Record(ctx, 0)
			e.sent.Add(ctx, 1)
			return nil
		}
		if attempt == 0 {
			select {
			case <-time.After(100 * time.Millisecond):
			case <-ctx.Done():
				err = ctx.Err()
			}
		}
		if ctx.Err() != nil {
			break
		}
	}
	e.failed.Add(ctx, 1)
	e.mu.Lock()
	e.failures++
	e.probing = false
	if e.failures >= 2 {
		e.openUntil = time.Now().Add(5 * time.Second)
		e.state.Record(context.Background(), 1)
	}
	e.mu.Unlock()
	return err
}

func (e *EventSenderAdapter) enqueue(ctx context.Context, event cloudevents.Event) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.queue) == queueLimit {
		e.dropped.Add(ctx, 1)
		e.logger.Warn("Kafka replay queue full; dropping newest event", "eventID", event.ID())
		return false
	}
	e.queue = append(e.queue, event)
	return true
}

// Replay retries queued events until the context is cancelled. Events remain in memory only.
func (e *EventSenderAdapter) Replay(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		e.mu.Lock()
		if len(e.queue) == 0 {
			e.mu.Unlock()
			continue
		}
		event := e.queue[0]
		e.mu.Unlock()
		if err := e.deliver(ctx, event); err != nil {
			continue
		}
		e.mu.Lock()
		e.queue = e.queue[1:]
		e.mu.Unlock()
	}
}

// Degraded reports an observed Kafka send failure or pending replay work.
func (e *EventSenderAdapter) Degraded() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.failures > 0 || len(e.queue) > 0
}

func newSource(serverID string) string {
	return "opampcommander/server/" + serverID
}
