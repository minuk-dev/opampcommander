// Package adapter wires the hexagonal adapter layer into Uber FX.
//
// It groups the inbound (primary), outbound (secondary), and shared (common)
// adapters into a single module:
//   - primary:   HTTP server, controllers, scheduler executor (driving side)
//   - secondary: MongoDB client/database and persistence adapters (driven side)
//   - common:    transports shared by both directions (messaging)
package adapter

import (
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter/common"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter/primary"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter/secondary"
)

// NewAdapterModules creates the adapter layer module. The database type selects
// the secondary persistence backend (MongoDB or in-memory standalone).
func NewAdapterModules(databaseType config.DatabaseType) fx.Option {
	return fx.Module(
		"adapter",
		primary.New(),
		secondary.New(databaseType),
		common.New(),
	)
}
