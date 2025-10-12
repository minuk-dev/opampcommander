package out

import (
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
)

// New creates a new module for output adapters.
func New() fx.Option {
	return fx.Module(
		"outport",
		fx.Provide(
			NewMongoDBClient,
			NewMongoDatabase,
			helper.AsHealthIndicator(NewMongoDBHealthIndicator),
			fx.Annotate(mongodb.NewAgentRepository, fx.As(new(domainport.AgentPersistencePort))),
			fx.Annotate(mongodb.NewAgentGroupRepository, fx.As(new(domainport.AgentGroupPersistencePort))),
			fx.Annotate(mongodb.NewCommandRepository, fx.As(new(domainport.CommandPersistencePort))),
		),
	)
}
