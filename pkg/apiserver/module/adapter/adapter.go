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

	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/adapter/common"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/adapter/primary"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/adapter/secondary"
)

// NewAdapterModules creates the adapter layer module.
func NewAdapterModules() fx.Option {
	return fx.Module(
		"adapter",
		primary.New(),
		secondary.New(),
		common.New(),
	)
}
