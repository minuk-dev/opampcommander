package secondary

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/liveness/failover"
	outkafka "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/messaging/kafka"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

type eventBackendHealthIndicator struct {
	sender agentport.ServerEventSenderPort
}

func newEventBackendHealthIndicator(sender agentport.ServerEventSenderPort) *eventBackendHealthIndicator {
	return &eventBackendHealthIndicator{sender: sender}
}

func (*eventBackendHealthIndicator) Name() string { return "EventBackend" }
func (*eventBackendHealthIndicator) Readiness(context.Context) healthcheck.Readiness {
	return healthcheck.Readiness{Ready: true, Reason: ""}
}
func (i *eventBackendHealthIndicator) Health(context.Context) healthcheck.Health {
	if sender, ok := i.sender.(*outkafka.EventSenderAdapter); ok && sender.Degraded() {
		return healthcheck.Health{Healthy: true, Degraded: true, Reason: "Kafka delivery failed or replay is pending"}
	}

	return healthcheck.Health{Healthy: true, Degraded: false, Reason: ""}
}

// MongoDBHealthIndicator is a health indicator for MongoDB.
var _ healthcheck.HealthIndicator = (*MongoDBHealthIndicator)(nil)

// MongoDBHealthIndicator is a health indicator for MongoDB.
type MongoDBHealthIndicator struct {
	client *mongo.Client
}

// NewMongoDBHealthIndicator creates a new MongoDBHealthIndicator.
func NewMongoDBHealthIndicator(client *mongo.Client) *MongoDBHealthIndicator {
	return &MongoDBHealthIndicator{
		client: client,
	}
}

// Name returns the name of the health indicator.
func (m *MongoDBHealthIndicator) Name() string {
	return "MongoDB"
}

// Readiness returns the readiness status of the MongoDB connection.
func (m *MongoDBHealthIndicator) Readiness(ctx context.Context) healthcheck.Readiness {
	err := m.client.Ping(ctx, nil)
	if err != nil {
		return healthcheck.Readiness{
			Ready:  false,
			Reason: err.Error(),
		}
	}

	return healthcheck.Readiness{
		Ready:  true,
		Reason: "",
	}
}

// Health returns the health status of the MongoDB connection.
func (m *MongoDBHealthIndicator) Health(context.Context) healthcheck.Health {
	return healthcheck.Health{
		Healthy:  true,
		Degraded: false,
		Reason:   "when mongodb is initialized, it's always healthy",
	}
}

// AgentLivenessHealthIndicator reports whether the shared agent-liveness fast tier
// is in use.
var _ healthcheck.HealthIndicator = (*AgentLivenessHealthIndicator)(nil)

// AgentLivenessHealthIndicator reports whether the shared agent-liveness fast tier
// is in use.
//
// It never fails a probe. The tier is a pure accelerator: losing it costs database
// writes, not correctness, and a failing probe would restart a process that is
// serving perfectly well. Its outage surfaces as degraded instead — visible in the
// health body, with the status code left at 200.
type AgentLivenessHealthIndicator struct {
	store *failover.Store
}

// NewAgentLivenessHealthIndicator creates the indicator.
func NewAgentLivenessHealthIndicator(store *failover.Store) *AgentLivenessHealthIndicator {
	return &AgentLivenessHealthIndicator{
		store: store,
	}
}

// Name returns the name of the health indicator.
func (i *AgentLivenessHealthIndicator) Name() string {
	return "AgentLivenessCache"
}

// Readiness always reports ready: the server serves agents whether or not the
// shared tier is reachable.
func (i *AgentLivenessHealthIndicator) Readiness(context.Context) healthcheck.Readiness {
	return healthcheck.Readiness{
		Ready:  true,
		Reason: "",
	}
}

// Health reports the tier as degraded while the circuit breaker is not routing to
// it, and healthy otherwise.
func (i *AgentLivenessHealthIndicator) Health(context.Context) healthcheck.Health {
	state := i.store.State()
	if state == failover.StateClosed {
		return healthcheck.Health{
			Healthy:  true,
			Degraded: false,
			Reason:   "",
		}
	}

	return healthcheck.Health{
		Healthy:  true,
		Degraded: true,
		Reason: fmt.Sprintf(
			"shared liveness tier unavailable (circuit breaker %s); "+
				"serving from node-local state and the database", state),
	}
}
