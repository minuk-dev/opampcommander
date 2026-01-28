// Package domain provides the domain services module for the API server.
package domain

import (
	"go.uber.org/fx"

	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	domainservice "github.com/minuk-dev/opampcommander/internal/domain/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
)

// New creates a new module for domain services.
func New() fx.Option {
	components := []any{
		fx.Annotate(domainservice.NewConnectionService, fx.As(new(domainport.ConnectionUsecase))),
		fx.Annotate(domainservice.NewAgentService, fx.As(new(domainport.AgentUsecase))),
		fx.Annotate(domainservice.NewAgentGroupService,
			fx.As(new(domainport.AgentGroupUsecase)),
			fx.As(new(domainport.AgentGroupRelatedUsecase)),
		),
		fx.Annotate(domainservice.NewAgentPackageService, fx.As(new(domainport.AgentPackageUsecase))),
		fx.Annotate(domainservice.NewAgentRemoteConfigService, fx.As(new(domainport.AgentRemoteConfigUsecase))),
		domainservice.NewServerService,
		fx.Annotate(
			Identity[*domainservice.ServerService],
			fx.As(new(domainport.ServerUsecase)),
			fx.As(new(domainport.ServerMessageUsecase)),
		),
		domainservice.NewServerIdentityService,
		fx.Annotate(
			Identity[*domainservice.ServerIdentityService],
			fx.As(new(domainport.ServerIdentityProvider)),
		),
		fx.Annotate(domainservice.NewAgentNotificationService, fx.As(new(domainport.AgentNotificationUsecase))),
		helper.AsRunner(Identity[*domainservice.ServerService]),
		helper.AsRunner(Identity[*domainservice.ServerIdentityService]),
	}

	return fx.Module(
		"domain",
		fx.Provide(components...),
	)
}

// Identity is a generic function that returns the input value.
// It is a helper function to generate a function that returns the input value.
// It is used to provide a function as a interface.
func Identity[T any](a T) T {
	return a
}
