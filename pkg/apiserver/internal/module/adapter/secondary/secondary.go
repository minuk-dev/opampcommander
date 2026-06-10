// Package secondary provides outbound (driven) adapters for the API server,
// such as the MongoDB client/database and the persistence repositories built on it.
package secondary

import (
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// New creates the secondary adapter module, selecting the persistence backend
// from the configured database type. "inmemory" wires the in-memory store
// (standalone mode); any other value (including the default "mongodb") wires the
// MongoDB client, database, and repositories.
func New(databaseType config.DatabaseType) fx.Option {
	persistence := NewMongoDB()
	if databaseType == config.DatabaseTypeInMemory {
		persistence = NewInMemory()
	}

	return fx.Options(
		persistence,
		// Outbound messaging: server-event sender.
		fx.Provide(newEventSender),
	)
}
