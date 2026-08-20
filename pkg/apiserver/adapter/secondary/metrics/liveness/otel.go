package liveness

import (
	"context"
	"fmt"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.AgentLivenessMetricsPort = (*OTelRecorder)(nil)

// meterName namespaces the instruments this adapter owns.
const meterName = "github.com/minuk-dev/opampcommander/agentliveness"

// Instrument names. They read as a set: absorbed and written are the two halves of
// every observation, and their difference is the database writes the fast tier saved.
const (
	absorbedMetric     = "opampcommander.agent_liveness.absorbed"
	writtenMetric      = "opampcommander.agent_liveness.written"
	fallbackMetric     = "opampcommander.agent_liveness.fallback"
	breakerStateMetric = "opampcommander.agent_liveness.breaker_state"
)

// Circuit breaker states, encoded for the gauge.
const (
	breakerClosed   int64 = 0
	breakerHalfOpen int64 = 1
	breakerOpen     int64 = 2
)

// OTelRecorder reports liveness measurements through OpenTelemetry.
type OTelRecorder struct {
	absorbed metric.Int64Counter
	written  metric.Int64Counter
	fallback metric.Int64Counter

	// breakerState is held rather than measured on demand because the breaker
	// changes state on the message path, where an observable callback cannot reach.
	breakerState atomic.Int64
}

// NewOTelRecorder creates a recorder over the given meter provider.
func NewOTelRecorder(meterProvider metric.MeterProvider) (*OTelRecorder, error) {
	meter := meterProvider.Meter(meterName)

	absorbed, err := meter.Int64Counter(absorbedMetric,
		metric.WithDescription(
			"Agent liveness observations absorbed by the fast tier without a database write. "+
				"The difference against "+writtenMetric+" is the database writes the fast tier saved."),
		metric.WithUnit("{observation}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the %s counter: %w", absorbedMetric, err)
	}

	written, err := meter.Int64Counter(writtenMetric,
		metric.WithDescription(
			"Agent liveness observations written through to the database, by write shape: "+
				"a full document rewrite, or a liveness-only update."),
		metric.WithUnit("{observation}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the %s counter: %w", writtenMetric, err)
	}

	fallback, err := meter.Int64Counter(fallbackMetric,
		metric.WithDescription(
			"Agent liveness operations served by the node-local fallback because the shared tier could not answer."),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the %s counter: %w", fallbackMetric, err)
	}

	//exhaustruct:ignore // the atomic starts at zero, which is the closed state
	recorder := &OTelRecorder{
		absorbed: absorbed,
		written:  written,
		fallback: fallback,
	}

	err = registerBreakerStateGauge(meter, recorder)
	if err != nil {
		return nil, err
	}

	return recorder, nil
}

// RecordHeartbeatAbsorbed implements [agentport.AgentLivenessMetricsPort].
func (r *OTelRecorder) RecordHeartbeatAbsorbed() {
	r.absorbed.Add(context.Background(), 1)
}

// RecordWriteThrough implements [agentport.AgentLivenessMetricsPort].
func (r *OTelRecorder) RecordWriteThrough(shape agentport.LivenessWriteShape) {
	r.written.Add(context.Background(), 1,
		metric.WithAttributes(attribute.String("shape", string(shape))))
}

// RecordFallback implements [agentport.AgentLivenessMetricsPort].
func (r *OTelRecorder) RecordFallback(operation string) {
	r.fallback.Add(context.Background(), 1,
		metric.WithAttributes(attribute.String("operation", operation)))
}

// RecordBreakerState implements [agentport.AgentLivenessMetricsPort].
func (r *OTelRecorder) RecordBreakerState(state string) {
	r.breakerState.Store(encodeBreakerState(state))
}

func registerBreakerStateGauge(meter metric.Meter, recorder *OTelRecorder) error {
	gauge, err := meter.Int64ObservableGauge(breakerStateMetric,
		metric.WithDescription(
			"Circuit breaker state for the shared agent liveness tier: 0 closed, 1 half-open, 2 open. "+
				"Reads 0 when no shared tier is configured — nothing to be degraded by."),
	)
	if err != nil {
		return fmt.Errorf("failed to create the %s gauge: %w", breakerStateMetric, err)
	}

	_, err = meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		observer.ObserveInt64(gauge, recorder.breakerState.Load())

		return nil
	}, gauge)
	if err != nil {
		return fmt.Errorf("failed to register the %s callback: %w", breakerStateMetric, err)
	}

	return nil
}

// encodeBreakerState maps a state name to its gauge value. An unknown name is
// reported as open: a state we cannot read is not one to treat as healthy.
func encodeBreakerState(state string) int64 {
	switch state {
	case "closed":
		return breakerClosed
	case "half-open":
		return breakerHalfOpen
	default:
		return breakerOpen
	}
}
