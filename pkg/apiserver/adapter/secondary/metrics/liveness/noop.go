// Package liveness provides the metrics adapters for the agent liveness fast tier.
package liveness

import (
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.AgentLivenessMetricsPort = (*NoopRecorder)(nil)

// NoopRecorder discards every measurement. It is wired when metrics are disabled,
// so the domain can always call the port without a nil check on the hot path.
type NoopRecorder struct{}

// NewNoopRecorder creates a recorder that discards every measurement.
func NewNoopRecorder() *NoopRecorder {
	return &NoopRecorder{}
}

// RecordHeartbeatAbsorbed implements [agentport.AgentLivenessMetricsPort].
func (*NoopRecorder) RecordHeartbeatAbsorbed() {}

// RecordWriteThrough implements [agentport.AgentLivenessMetricsPort].
func (*NoopRecorder) RecordWriteThrough(agentport.LivenessWriteShape) {}

// RecordFallback implements [agentport.AgentLivenessMetricsPort].
func (*NoopRecorder) RecordFallback(string) {}

// RecordBreakerState implements [agentport.AgentLivenessMetricsPort].
func (*NoopRecorder) RecordBreakerState(string) {}
