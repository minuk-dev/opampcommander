// Package domain provides the domain services module for the API server.
package domain

import (
	"go.uber.org/fx"

	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	domainservice "github.com/minuk-dev/opampcommander/internal/domain/service"
)

// New creates a new module for domain services.
func New() fx.Option {
	components := []any{
		fx.Annotate(domainservice.NewCommandService, fx.As(new(domainport.CommandUsecase))),
		fx.Annotate(domainservice.NewConnectionService, fx.As(new(domainport.ConnectionUsecase))),
		fx.Annotate(domainservice.NewAgentService, fx.As(new(domainport.AgentUsecase))),
		fx.Annotate(domainservice.NewAgentGroupService,
			fx.As(new(domainport.AgentGroupUsecase)),
			fx.As(new(domainport.AgentGroupRelatedUsecase)),
		),
	}

	return fx.Module(
		"domain",
		fx.Provide(components...),
	)
}
