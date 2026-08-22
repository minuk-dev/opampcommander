package secondary

import (
	"go.uber.org/fx"

	livenessinmemory "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/liveness/inmemory"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// NewLiveness provides the agent liveness fast tier.
//
// Today that is always the node-local store, which reproduces the per-node view
// of agent liveness the OpAMP service used to keep in a process-local map. It is
// provided as a concrete singleton as well as through the port so the scheduler
// executor can run its GC sweep.
func NewLiveness() fx.Option {
	return fx.Options(
		fx.Provide(
			newInMemoryLivenessStore,
			fx.Annotate(identity[*livenessinmemory.Store], fx.As(new(agentport.AgentLivenessPort))),
			helper.AsRunner(identity[*livenessinmemory.Store]),
		),
	)
}

func newInMemoryLivenessStore() *livenessinmemory.Store {
	//exhaustruct:ignore // zero fields select the store's defaults
	return livenessinmemory.New(livenessinmemory.Config{}, clock.NewRealClock())
}
