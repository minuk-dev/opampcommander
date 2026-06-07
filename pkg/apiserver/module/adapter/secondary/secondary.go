// Package secondary provides outbound (driven) adapters for the API server,
// such as the MongoDB client/database and the persistence repositories built on it.
package secondary

import (
	"go.uber.org/fx"
)

// New creates the secondary adapter module.
func New() fx.Option {
	return fx.Options(
		NewMongoDB(),
		// Outbound messaging: server-event sender.
		fx.Provide(newEventSender),
	)
}
