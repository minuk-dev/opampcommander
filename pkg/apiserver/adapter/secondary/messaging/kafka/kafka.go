// Package kafka implements Kafka messaging adapters.
package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
	observabilityClient "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/client"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	kafkamodel "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/kafka"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
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
	settings  config.KafkaSettings
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
	protocolSender *Sender,
	logger *slog.Logger,
	meterProvider metric.MeterProvider,
	settings config.KafkaSettings,
) (*EventSenderAdapter, error) {
	settings = settings.WithDefaults()

	err := settings.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid Kafka delivery settings: %w", err)
	}

	//nolint:godox
	// TODO: cloudevents's observability does not support to inject TracerProvider instead of global
	// https://github.com/cloudevents/sdk-go/pull/1202
	otelService := observabilityClient.NewOTelObservabilityService()

	sender, err := cloudevents.NewClient(protocolSender, client.WithObservabilityService(otelService))
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudEvents client for sender: %w", err)
	}

	if meterProvider == nil {
		meterProvider = noop.NewMeterProvider()
	}

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
		settings:  settings,
		send:      func(ctx context.Context, event cloudevents.Event) error { return sender.Send(ctx, event) },
		logger:    logger,
		clock:     clock.NewRealClock(),
		mu:        sync.Mutex{},
		queue:     nil,
		failures:  0,
		openUntil: time.Time{},
		probing:   false,
		sent:      sent, failed: failed, dropped: dropped, state: state,
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
		if permanentError(err) {
			e.drop(ctx, event, err)

			return fmt.Errorf("send event for server %s: %w", serverID, err)
		}

		if ctx.Err() != nil {
			return fmt.Errorf("send message cancelled: %w", ctx.Err())
		}

		if !e.enqueue(ctx, event) {
			return fmt.Errorf("server %s: %w: replay queue full (event dropped)", serverID, model.ErrTargetServerUnreachable)
		}

		return fmt.Errorf("server %s: %w: %w", serverID, model.ErrTargetServerUnreachable, err)
	}

	return nil
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

		err := e.deliver(ctx, event)
		if err != nil && !permanentError(err) {
			continue
		}

		if err != nil {
			e.drop(ctx, event, err)
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

// deliver bounds each request and opens the breaker at the configured failure threshold.
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

	ctx, cancel := context.WithTimeout(ctx, e.settings.SendTimeout)
	defer cancel()

	var err error

	for attempt := range e.settings.RetryAttempts {
		err = e.send(ctx, event)
		if err == nil {
			e.mu.Lock()
			e.failures = 0
			e.openUntil = time.Time{}
			e.probing = false
			e.mu.Unlock()
			e.state.Record(ctx, 0)
			e.sent.Add(ctx, 1)

			return nil
		}

		if permanentError(err) {
			e.recordFailure(ctx, err)

			return fmt.Errorf("permanent Kafka send failure: %w", err)
		}

		if attempt+1 < e.settings.RetryAttempts {
			select {
			case <-time.After(e.settings.RetryBackoff):
			case <-ctx.Done():
				err = ctx.Err()
			}
		}

		if ctx.Err() != nil {
			break
		}
	}

	e.recordFailure(ctx, err)

	return fmt.Errorf("kafka send failed: %w", err)
}

func (e *EventSenderAdapter) recordFailure(ctx context.Context, err error) {
	e.failed.Add(ctx, 1)
	e.mu.Lock()
	defer e.mu.Unlock()

	e.probing = false

	if permanentError(err) {
		return
	}

	e.failures++
	if e.failures >= e.settings.FailureThreshold {
		e.openUntil = time.Now().Add(e.settings.ProbeInterval)
		e.state.Record(ctx, 1)
	}
}

func (e *EventSenderAdapter) enqueue(ctx context.Context, event cloudevents.Event) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.queue) >= e.settings.QueueLimit {
		e.dropped.Add(ctx, 1)
		e.logger.Warn("Kafka replay queue full; dropping newest event", "eventID", event.ID())

		return false
	}

	e.queue = append(e.queue, event)

	return true
}

func (e *EventSenderAdapter) drop(ctx context.Context, event cloudevents.Event, err error) {
	e.dropped.Add(ctx, 1)
	e.logger.Error("dropping Kafka event after permanent failure", "eventID", event.ID(), "error", err)
}

func permanentError(err error) bool {
	var configurationError sarama.ConfigurationError

	return errors.Is(err, errEventEncoding) || errors.As(err, &configurationError) ||
		errors.Is(err, sarama.ErrMessageSizeTooLarge) ||
		errors.Is(err, sarama.ErrMessageSetSizeTooLarge) || errors.Is(err, sarama.ErrInvalidTopic) ||
		errors.Is(err, sarama.ErrTopicAuthorizationFailed) || errors.Is(err, sarama.ErrClusterAuthorizationFailed) ||
		errors.Is(err, sarama.ErrUnsupportedVersion) || errors.Is(err, sarama.ErrInvalidRecord)
}

func newSource(serverID string) string {
	return "opampcommander/server/" + serverID
}
