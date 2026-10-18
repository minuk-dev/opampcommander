package kafka

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

var _ healthcheck.HealthIndicator = (*Sender)(nil)

// Observe exports notification outcomes without waiting for acknowledgements in Send.
func (s *Sender) Observe(meter metric.Meter) error {
	events, err := meter.Int64ObservableCounter("opampcommander.kafka.events",
		metric.WithDescription("Kafka notifications by outcome: enqueued, delivered to the broker, "+
			"failed after retries, or rejected before enqueue."))
	if err != nil {
		return fmt.Errorf("create Kafka events counter: %w", err)
	}

	degraded, err := meter.Int64ObservableGauge("opampcommander.kafka.delivery_degraded",
		metric.WithDescription("1 when the last observed producer result failed; 0 otherwise. "+
			"Does not probe idle brokers or consumer health."))
	if err != nil {
		return fmt.Errorf("create Kafka delivery gauge: %w", err)
	}

	_, err = meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		for result, value := range map[string]int64{
			"enqueued": s.enqueued.Load(), "delivered": s.delivered.Load(),
			"failed": s.failed.Load(), "rejected": s.rejected.Load(),
		} {
			observer.ObserveInt64(events, value, metric.WithAttributes(attribute.String("outcome", result)))
		}

		var state int64
		if s.lastFailure.Load() != nil {
			state = 1
		}

		observer.ObserveInt64(degraded, state)

		return nil
	}, events, degraded)
	if err != nil {
		return fmt.Errorf("register Kafka delivery metrics: %w", err)
	}

	return nil
}

// Name identifies the producer separately from broker reachability and consumer health.
func (s *Sender) Name() string { return "KafkaProducer" }

// Readiness stays ready because persisted state and local agent delivery still work.
func (s *Sender) Readiness(context.Context) healthcheck.Readiness {
	return healthcheck.Readiness{Ready: true, Reason: ""}
}

// Health reports the last observed delivery failure without failing liveness.
func (s *Sender) Health(context.Context) healthcheck.Health {
	health := healthcheck.Health{Healthy: true, Degraded: false, Reason: ""}
	if failure := s.lastFailure.Load(); failure != nil {
		health.Degraded = true
		health.Reason = "last Kafka notification delivery failed; heartbeat recovery remains available: " + *failure
	}

	return health
}
