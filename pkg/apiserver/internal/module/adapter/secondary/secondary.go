// Package secondary provides outbound (driven) adapters for the API server,
// such as the MongoDB client/database and the persistence repositories built on it.
package secondary

import (
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/cached"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

// New creates the secondary adapter module, selecting the persistence backend
// from the configured database type. Only the explicit "mongodb" wires the
// MongoDB client, database, and repositories; any other value (including the
// default/empty) wires the in-memory store (standalone mode).
func New(databaseType config.DatabaseType, livenessSettings config.LivenessSettings) fx.Option {
	persistence := NewInMemory()
	if databaseType == config.DatabaseTypeMongoDB {
		persistence = NewMongoDB()
	}

	return fx.Options(
		persistence,
		// Live sockets and their session state are local to every server,
		// regardless of the durable persistence backend.
		fx.Provide(fx.Annotate(inmemory.NewConnectionStore, fx.As(new(agentport.ConnectionStore)))),
		fx.Provide(
			provideAgentStore,
			provideServerStore,
			fx.Annotate(identity[*cached.AgentStore], fx.As(new(agentport.AgentStore))),
			fx.Annotate(identity[*cached.ServerStore], fx.As(new(agentport.ServerStore))),
			fx.Annotate(provideNotificationStore, fx.As(new(agentport.NotificationStore))),
			fx.Annotate(provideAgentGroupChangeStore, fx.As(new(agentport.AgentGroupChangeStore))),
		),
		fx.Invoke(registerStoreShutdown),
		// Agent liveness fast tier (node-local by default).
		NewLiveness(databaseType, livenessSettings),
		// Outbound messaging: server-event sender.
		fx.Provide(newEventSender),
		// Outbound metrics: endpoint-throughput query port (Prometheus or no-op).
		fx.Provide(newEndpointMetricsQueryAdapter),
	)
}
